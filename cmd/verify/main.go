// Command verify is a live smoke test for the spoapi CSOM transport: it connects
// app-only and reads the tenant admin object (Get-SPOTenant) via the generated
// bindings — proving the ProcessQuery wire is accepted by the real service and the
// response decodes (incl. enum normalisation). Read-only.
//
// Credentials come from the ARM_*/AZURE_* environment via go-msadmin/authx; the
// SharePoint admin endpoint from SPO_ADMIN_URL or SPO_TENANT_NAME. App-only over
// CSOM needs SharePoint Sites.FullControl.All and a *certificate* credential
// (SharePoint rejects client-secret app-only tokens):
//
//	ARM_TENANT_ID=… ARM_CLIENT_ID=… ARM_CLIENT_CERTIFICATE_PATH=… \
//	  SPO_TENANT_NAME=contoso go run ./cmd/verify
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/terraprovider/go-msadmin/authx"
	"github.com/terraprovider/go-spo/spo"
	"github.com/terraprovider/go-spo/spoapi"
)

func main() {
	adminURL := os.Getenv("SPO_ADMIN_URL")
	tenant := os.Getenv("SPO_TENANT_NAME")
	if adminURL == "" && tenant == "" {
		fmt.Fprintln(os.Stderr, "set SPO_ADMIN_URL or SPO_TENANT_NAME")
		os.Exit(2)
	}

	tp, err := authx.FromEnv().Build()
	if err != nil {
		fmt.Fprintln(os.Stderr, "auth:", err)
		os.Exit(2)
	}
	c, err := spoapi.New(spoapi.Options{
		AdminURL:   adminURL,
		TenantName: tenant,
		Tokens:     tp,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "new:", err)
		os.Exit(1)
	}

	ctx := context.Background()
	svc := spo.New(c)
	fmt.Printf("connecting to %s\n", c.AdminURL())

	// 1) Tenant singleton (constructor object path).
	res, err := svc.GetSPOTenant(ctx, spo.GetSPOTenantParams{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ✗ Get-SPOTenant: %v\n", err)
		os.Exit(1)
	}
	obj := res.First()
	if obj == nil {
		fmt.Fprintln(os.Stderr, "  ✗ Get-SPOTenant returned no object")
		os.Exit(1)
	}
	fmt.Printf("  ✓ Get-SPOTenant → %d properties\n", len(obj))
	for _, k := range []string{"SharingCapability", "OneDriveStorageQuota", "SharingDomainRestrictionMode"} {
		if v, ok := obj[k]; ok {
			fmt.Printf("    %-32s %v\n", k, v)
		}
	}

	// 2) A site collection by URL (method object path — the generalized transport).
	root := strings.Replace(c.AdminURL(), "-admin.", ".", 1)
	sres, err := svc.GetSPOSite(ctx, spo.GetSPOSiteParams{Identity: root})
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ✗ Get-SPOSite %s: %v\n", root, err)
		os.Exit(1)
	}
	site := sres.First()
	if site == nil {
		fmt.Fprintln(os.Stderr, "  ✗ Get-SPOSite returned no object")
		os.Exit(1)
	}
	fmt.Printf("  ✓ Get-SPOSite %s → %d properties\n", root, len(site))
	for _, k := range []string{"Title", "Url", "Status", "StorageMaximumLevel", "LockState", "SharingCapability", "Template"} {
		if v, ok := site[k]; ok {
			fmt.Printf("    %-32s %v\n", k, v)
		}
	}
}
