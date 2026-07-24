package spo

import (
	"context"
	"strings"

	"github.com/terraprovider/go-spo/spoapi"
)

// NewSPOOrgNewsSiteParams designates a site as an organization news site (keyed by URL).
type NewSPOOrgNewsSiteParams struct{ Identity string }

// NewSPOOrgNewsSite designates a site as an org news site (Tenant.SetOrgNewsSite(url)).
func (s *Service) NewSPOOrgNewsSite(ctx context.Context, p NewSPOOrgNewsSiteParams) (*spoapi.Result, error) {
	return s.C.Invoke(ctx, spoapi.Op{
		CmdletName: "Set-SPOOrgNewsSite",
		TypeID:     spoapi.TenantTypeID,
		Invoke:     []spoapi.MethodCall{{Name: "SetOrgNewsSite", Params: []spoapi.Param{{Type: spoapi.TypeString, Value: p.Identity}}}},
	})
}

// GetSPOOrgNewsSiteParams targets an org news site by URL.
type GetSPOOrgNewsSiteParams struct{ Identity string }

// GetSPOOrgNewsSite reports whether the URL is an org news site (Tenant.GetOrgNewsSites()).
func (s *Service) GetSPOOrgNewsSite(ctx context.Context, p GetSPOOrgNewsSiteParams) (*spoapi.Result, error) {
	sites, err := s.C.MethodStrings(ctx, "Get-SPOOrgNewsSite", spoapi.TenantTypeID, "GetOrgNewsSites", nil)
	if err != nil {
		return nil, err
	}
	for _, u := range sites {
		if strings.EqualFold(strings.TrimRight(u, "/"), strings.TrimRight(p.Identity, "/")) {
			return &spoapi.Result{Value: []map[string]any{{"Identity": p.Identity, "Id": p.Identity, "Url": u}}}, nil
		}
	}
	return nil, &spoapi.APIError{Status: 404, Message: "org news site " + p.Identity + " not found"}
}

// RemoveSPOOrgNewsSiteParams targets an org news site by URL.
type RemoveSPOOrgNewsSiteParams struct{ Identity string }

// RemoveSPOOrgNewsSite removes the org-news designation (Tenant.RemoveOrgNewsSite(url)).
func (s *Service) RemoveSPOOrgNewsSite(ctx context.Context, p RemoveSPOOrgNewsSiteParams) (*spoapi.Result, error) {
	return s.C.Invoke(ctx, spoapi.Op{
		CmdletName: "Remove-SPOOrgNewsSite",
		TypeID:     spoapi.TenantTypeID,
		Invoke:     []spoapi.MethodCall{{Name: "RemoveOrgNewsSite", Params: []spoapi.Param{{Type: spoapi.TypeString, Value: p.Identity}}}},
	})
}
