// Package spoapi is a lean Go client for the SharePoint Online tenant/site admin
// API — the CSOM (Client-Side Object Model) surface behind the SharePoint Online
// Management Shell (Microsoft.Online.SharePoint.PowerShell / Connect-SPOService).
//
// Unlike the REST-based sibling clients (Teams, Exchange), SharePoint admin is
// CSOM: the client batches actions against object paths as XML and POSTs them to a
// single endpoint — /_vti_bin/client.svc/ProcessQuery on the tenant admin host —
// and the server returns a JSON array of results. Errors are returned inside an
// HTTP 200 as an ErrorInfo object. See spo-powershell-api-re/docs/02-csom-protocol.md.
//
// The core is intentionally thin: transport pooling and retry are delegated to the
// *http.Client you pass in (wrap it with go-msadmin/retry). Auth is abstracted
// behind TokenProvider (go-msadmin/auth); the token audience is the tenant admin
// URL itself.
//
// NOTE: SharePoint does not support client-secret app-only auth. The CSOM endpoint
// rejects secret-based app tokens with 401 "Unsupported app only token" (it requires
// appidacr=2 — a certificate credential), before any permission check. App-only must
// use a certificate; a client secret is not a valid credential for this API.
package spoapi

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/terraprovider/go-msadmin/auth"
	"github.com/terraprovider/go-msadmin/retry"
)

// Environment selects the SharePoint host suffix for a national cloud. The token
// audience and admin host are derived per-tenant from TenantName + this suffix
// (spo-powershell-api-re/spec/environments.json).
type Environment struct {
	Name      string // "Commercial", "GCCHigh", "DoD", "China"
	SPOSuffix string // e.g. "sharepoint.com" → https://{tenant}-admin.sharepoint.com
	Authority string // AAD login host (informational; the TokenProvider authenticates)
}

var (
	// Commercial is the worldwide/GCC cloud.
	Commercial = Environment{Name: "Commercial", SPOSuffix: "sharepoint.com", Authority: "https://login.microsoftonline.com"}
	// GCCHigh is US Government Community Cloud High.
	GCCHigh = Environment{Name: "GCCHigh", SPOSuffix: "sharepoint.us", Authority: "https://login.microsoftonline.us"}
	// DoD is US Department of Defense.
	DoD = Environment{Name: "DoD", SPOSuffix: "sharepoint-mil.us", Authority: "https://login.microsoftonline.us"}
	// China is the 21Vianet cloud.
	China = Environment{Name: "China", SPOSuffix: "sharepoint.cn", Authority: "https://login.chinacloudapi.cn"}
)

// TokenProvider is the shared token abstraction (go-msadmin/auth): it returns a
// bearer token whose audience is the tenant admin URL. App-only requires a
// certificate credential (SharePoint rejects client-secret app-only tokens) with
// the SharePoint Sites.FullControl.All application permission.
type TokenProvider = auth.TokenProvider

// StaticTokenProvider serves a fixed pre-acquired JWT (handy for tests/scripts).
type StaticTokenProvider = auth.StaticToken

// DefaultApplicationName is the ProcessQuery <Request ApplicationName="…">. The
// CSOM runtime default is ".NET Library"; refined to the module's value once
// confirmed by live capture.
const DefaultApplicationName = ".NET Library"

// DefaultUserAgent mimics the SharePoint Online Management Shell. The exact string
// (and module version) is confirmed by live capture; refine per module release.
const DefaultUserAgent = "SharePoint Online Management Shell"

// Options configures a Client. Provide either TenantName (+ Environment) or an
// explicit AdminURL.
type Options struct {
	Environment Environment   // defaults to Commercial
	TenantName  string        // e.g. "contoso" → https://contoso-admin.sharepoint.com
	AdminURL    string        // explicit admin URL override (wins over TenantName)
	Tokens      TokenProvider // required
	HTTPClient  *http.Client  // optional; wrap with go-msadmin/retry. A retrying client is used if nil.

	ApplicationName string // wire-fidelity; defaults to DefaultApplicationName
	UserAgent       string // wire-fidelity; defaults to DefaultUserAgent
}

// Client talks to one tenant's SharePoint admin host. Safe for concurrent use.
type Client struct {
	opt       Options
	http      *http.Client
	adminURL  string // https://{tenant}-admin.{suffix} (no trailing slash)
	resource  string // token audience = adminURL
	appName   string
	userAgent string
}

// New builds a Client. Tokens and (TenantName or AdminURL) are required.
func New(opt Options) (*Client, error) {
	if opt.Tokens == nil {
		return nil, fmt.Errorf("spoapi: Options.Tokens is required")
	}
	if opt.Environment.SPOSuffix == "" {
		opt.Environment = Commercial
	}
	admin := strings.TrimRight(opt.AdminURL, "/")
	if admin == "" {
		if opt.TenantName == "" {
			return nil, fmt.Errorf("spoapi: Options.TenantName or Options.AdminURL is required")
		}
		admin = "https://" + opt.TenantName + "-admin." + opt.Environment.SPOSuffix
	}
	if opt.HTTPClient == nil {
		// SharePoint throttles with 429/Retry-After and returns transient 5xx under
		// load; default to the go-msadmin retry transport. Callers can override.
		opt.HTTPClient = &http.Client{Transport: retry.NewTransport(nil, retry.Config{})}
	}
	if opt.ApplicationName == "" {
		opt.ApplicationName = DefaultApplicationName
	}
	if opt.UserAgent == "" {
		opt.UserAgent = DefaultUserAgent
	}
	return &Client{
		opt:       opt,
		http:      opt.HTTPClient,
		adminURL:  admin,
		resource:  admin,
		appName:   opt.ApplicationName,
		userAgent: opt.UserAgent,
	}, nil
}

// AdminURL returns the resolved tenant admin URL.
func (c *Client) AdminURL() string { return c.adminURL }

// processQueryURL is the single CSOM endpoint every op targets.
func (c *Client) processQueryURL() string {
	return c.adminURL + "/_vti_bin/client.svc/ProcessQuery"
}
