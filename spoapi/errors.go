package spoapi

import (
	"net/http"
	"strings"

	"github.com/terraprovider/go-msadmin/httpx"
)

// APIError is the shared error type (go-msadmin/httpx) for a failed call.
type APIError = httpx.APIError

// IsNotFound reports whether err is a not-found from SharePoint (a missing site,
// etc.). Terraform providers wire their isNotFound helper to this.
func IsNotFound(err error) bool { return httpx.IsNotFound(err) }

// newCSOMError converts a ProcessQuery ErrorInfo (returned inside an HTTP 200) into
// an *APIError. SharePoint has no HTTP status for these, so we synthesise one:
// not-found conditions map to 404 (so IsNotFound works), everything else to 400.
func newCSOMError(e *csomErrorInfo, raw []byte) error {
	status := http.StatusBadRequest
	if isNotFound(e) {
		status = http.StatusNotFound
	}
	return &APIError{
		Status:  status,
		Code:    e.ErrorTypeName,
		Message: strings.TrimSpace(e.ErrorMessage),
		Body:    truncate(raw, 2000),
	}
}

// isNotFound heuristically classifies a CSOM error as not-found. SharePoint does
// not use a single ErrorTypeName for this, so we also inspect the message. Refined
// against live captures (spo-powershell-api-re live-capture runbook).
func isNotFound(e *csomErrorInfo) bool {
	t := e.ErrorTypeName
	if strings.Contains(t, "NotFound") ||
		strings.Contains(t, "FileNotFoundException") ||
		strings.Contains(t, "ItemNotFoundException") ||
		t == "Microsoft.Online.SharePoint.Common.SpoNoSuchSiteException" {
		return true
	}
	m := strings.ToLower(e.ErrorMessage)
	return strings.Contains(m, "does not exist") ||
		strings.Contains(m, "cannot get site") ||
		strings.Contains(m, "cannot find") ||
		strings.Contains(m, "could not be found")
}
