package spoapi

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// csomHeader is element 0 of a ProcessQuery response array.
type csomHeader struct {
	SchemaVersion      string         `json:"SchemaVersion"`
	LibraryVersion     string         `json:"LibraryVersion"`
	ErrorInfo          *csomErrorInfo `json:"ErrorInfo"`
	TraceCorrelationID string         `json:"TraceCorrelationId"`
}

// csomErrorInfo is the error object SharePoint returns *inside* an HTTP 200.
type csomErrorInfo struct {
	ErrorMessage       string `json:"ErrorMessage"`
	ErrorValue         any    `json:"ErrorValue"`
	ErrorCode          int    `json:"ErrorCode"`
	ErrorTypeName      string `json:"ErrorTypeName"`
	TraceCorrelationID string `json:"TraceCorrelationId"`
}

// parseResponse decodes a ProcessQuery response array. On a CSOM error it returns
// an *httpx.APIError (built from ErrorInfo). Otherwise it returns the object
// produced by the <Query> action identified by queryID (nil for a write with no
// read-back).
//
// Shape: [ {header}, id1, result1, id2, result2, … ] — odd indices are action Ids,
// even indices their results.
// parseResults decodes a ProcessQuery response into a map of action-id → raw
// result, after surfacing any header ErrorInfo as an *APIError.
func parseResults(raw []byte) (map[int]json.RawMessage, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, nil
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, fmt.Errorf("spoapi: response is not a CSOM array: %w (body: %s)", err, truncate(raw, 300))
	}
	if len(arr) == 0 {
		return nil, nil
	}
	var hdr csomHeader
	if err := json.Unmarshal(arr[0], &hdr); err != nil {
		return nil, fmt.Errorf("spoapi: decode response header: %w", err)
	}
	if hdr.ErrorInfo != nil {
		return nil, newCSOMError(hdr.ErrorInfo, raw)
	}
	out := map[int]json.RawMessage{}
	for i := 1; i+1 < len(arr); i += 2 {
		var id int
		if err := json.Unmarshal(arr[i], &id); err != nil {
			continue // not an action-id slot; skip
		}
		out[id] = arr[i+1]
	}
	return out, nil
}

// parseResponse returns the object produced by the <Query> action queryID (or nil
// for a write with no read-back / a CSOM error).
func parseResponse(raw []byte, queryID int) ([]map[string]any, error) {
	results, err := parseResults(raw)
	if err != nil {
		return nil, err
	}
	if queryID == 0 || results == nil {
		return nil, nil
	}
	rm, ok := results[queryID]
	if !ok {
		return nil, nil
	}
	var obj map[string]any
	if err := json.Unmarshal(rm, &obj); err != nil {
		return nil, fmt.Errorf("spoapi: decode query result: %w", err)
	}
	return []map[string]any{obj}, nil
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "…"
}
