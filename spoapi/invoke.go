package spoapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/terraprovider/go-msadmin/httpx"
)

func debugEnabled() bool { return os.Getenv("SPOAPI_DEBUG") != "" }

// Result carries the objects returned by a ProcessQuery <Query> action, normalised
// to the property-bag shape the Terraform runtime (tf-msadmin/genframework)
// expects. A pure write with no read-back yields an empty Value.
type Result struct {
	Value []map[string]any
}

// Decode unmarshals Value into v (round-trips through JSON), so v's fields need
// json tags matching the CSOM property names.
func (r *Result) Decode(v any) error {
	b, err := json.Marshal(r.Value)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// First returns the first returned object (typical for singleton reads), or nil.
func (r *Result) First() map[string]any {
	if len(r.Value) == 0 {
		return nil
	}
	return r.Value[0]
}

// ChildItems returns the elements of a collection read (a <ChildItemQuery> result
// carries them under "_Child_Items_"), or nil for a non-collection result.
func (r *Result) ChildItems() []map[string]any {
	o := r.First()
	if o == nil {
		return nil
	}
	items, _ := o["_Child_Items_"].([]any)
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		if m, ok := it.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

// DecodeGuid strips the CSOM "/Guid(…)/" wrapper SharePoint uses to JSON-encode a
// GUID, returning the bare value; a plain string passes through unchanged.
func DecodeGuid(v any) string {
	s, _ := v.(string)
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "/Guid(") && strings.HasSuffix(s, ")/") {
		s = s[len("/Guid(") : len(s)-len(")/")]
	}
	return strings.Trim(s, "{}")
}

// Invoke builds op into a ProcessQuery request, POSTs it to the tenant admin
// ProcessQuery endpoint, and returns the resulting objects. CSOM errors (returned
// inside an HTTP 200 as ErrorInfo) surface as *APIError.
func (c *Client) Invoke(ctx context.Context, op Op) (*Result, error) {
	xml, queryID, err := buildRequest(op, c.appName)
	if err != nil {
		return nil, err
	}
	return c.invokeXML(ctx, op.CmdletName, xml, queryID)
}

// invokeXML POSTs a pre-built ProcessQuery body and parses the response. queryID is
// the <Query> action's id whose result to return (0 for none).
func (c *Client) invokeXML(ctx context.Context, cmdletName, xml string, queryID int) (*Result, error) {
	token, err := c.opt.Tokens.Token(ctx, c.resource)
	if err != nil {
		return nil, fmt.Errorf("spoapi: token: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.processQueryURL(), strings.NewReader(xml))
	if err != nil {
		return nil, err
	}
	c.setHeaders(req, Op{CmdletName: cmdletName}, token)
	if debugEnabled() {
		fmt.Fprintf(os.Stderr, "[spoapi] POST %s cmdlet=%q\n[spoapi] >>> %s\n", c.processQueryURL(), cmdletName, xml)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	raw, err := httpx.DecodeBody(resp)
	if err != nil {
		return nil, err
	}
	if debugEnabled() {
		fmt.Fprintf(os.Stderr, "[spoapi] <<< %d %s\n", resp.StatusCode, truncate(raw, 800))
	}
	if resp.StatusCode >= 400 {
		return nil, httpError(resp.StatusCode, raw)
	}
	value, err := parseResponse(raw, queryID)
	if err != nil {
		return nil, err
	}
	return &Result{Value: value}, nil
}

// setHeaders applies the ProcessQuery request headers. The CSOM request body is
// text/xml; the response is JSON. The User-Agent mimics the module for wire
// fidelity (refine the exact string via live capture).
func (c *Client) setHeaders(req *http.Request, op Op, token string) {
	h := req.Header
	h.Set("Authorization", "Bearer "+token)
	h.Set("Content-Type", "text/xml")
	h.Set("Accept", "*/*")
	h.Set("Accept-Encoding", "gzip, deflate, br")
	// A fidelity marker the SPO shell sends; confirm exact value via live capture.
	h.Set("X-ClientService-ClientTag", "SPO Management Shell")
	if c.userAgent != "" {
		h.Set("User-Agent", c.userAgent)
	}
	_ = op // op reserved for future per-cmdlet header fidelity
}

// httpError builds an APIError for a non-2xx HTTP response (auth/transport level;
// CSOM application errors arrive as 200 + ErrorInfo and are handled separately).
func httpError(status int, raw []byte) error {
	msg := strings.TrimSpace(truncate(raw, 600))
	if msg == "" {
		msg = http.StatusText(status)
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		msg += " — check the app has SharePoint Sites.FullControl.All (app-only needs a certificate credential; client secrets are rejected by SharePoint)"
	}
	return &APIError{Status: status, Message: msg, Body: truncate(raw, 2000)}
}
