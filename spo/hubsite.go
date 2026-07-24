package spo

import (
	"context"

	"github.com/terraprovider/go-spo/spoapi"
)

func hubChain(url string) []spoapi.PathStep {
	return []spoapi.PathStep{{Method: "GetHubSitePropertiesByUrl", Params: []spoapi.Param{{Type: spoapi.TypeString, Value: url}}}}
}

// hubSetProps builds the SetProperty list for the provided hub-site fields.
func hubSetProps(title, description, logoURL string, requiresJoinApproval bool) []spoapi.SetProp {
	return []spoapi.SetProp{
		{Name: "Title", Type: spoapi.TypeString, Value: title},
		{Name: "Description", Type: spoapi.TypeString, Value: description},
		{Name: "LogoUrl", Type: spoapi.TypeString, Value: logoURL},
		{Name: "RequiresJoinApproval", Type: spoapi.TypeBoolean, Value: requiresJoinApproval},
	}
}

// NewSPOHubSiteParams register an existing site as a hub (Register-SPOHubSite). The
// Identity is the site URL to promote.
type NewSPOHubSiteParams struct {
	Identity             string
	Title                string
	Description          string
	LogoUrl              string
	RequiresJoinApproval bool
}

// NewSPOHubSite registers a site as a hub (Tenant.RegisterHubSite(url)) and applies
// the given properties.
func (s *Service) NewSPOHubSite(ctx context.Context, p NewSPOHubSiteParams) (*spoapi.Result, error) {
	if _, err := s.C.Invoke(ctx, spoapi.Op{
		CmdletName: "Register-SPOHubSite",
		TypeID:     spoapi.TenantTypeID,
		Chain:      []spoapi.PathStep{{Method: "RegisterHubSite", Params: []spoapi.Param{{Type: spoapi.TypeString, Value: p.Identity}}}},
	}); err != nil {
		return nil, err
	}
	return s.SetSPOHubSite(ctx, SetSPOHubSiteParams(p))
}

// GetSPOHubSiteParams targets a hub site by its site URL.
type GetSPOHubSiteParams struct{ Identity string }

// GetSPOHubSite reads a hub site (Tenant.GetHubSitePropertiesByUrl(url)).
func (s *Service) GetSPOHubSite(ctx context.Context, p GetSPOHubSiteParams) (*spoapi.Result, error) {
	res, err := s.C.Invoke(ctx, spoapi.Op{
		CmdletName: "Get-SPOHubSite",
		TypeID:     spoapi.TenantTypeID,
		Chain:      hubChain(p.Identity),
		Query:      true,
	})
	if err != nil {
		return nil, err
	}
	o := res.First()
	if o == nil || isNull(o) {
		return nil, &spoapi.APIError{Status: 404, Message: "hub site " + p.Identity + " not found"}
	}
	o["Id"] = p.Identity
	o["Identity"] = p.Identity
	if id := spoapi.DecodeGuid(o["ID"]); id != "" {
		o["HubSiteId"] = id
	}
	return res, nil
}

// SetSPOHubSiteParams update a hub site's properties (Set-SPOHubSite -Identity <url>).
type SetSPOHubSiteParams struct {
	Identity             string
	Title                string
	Description          string
	LogoUrl              string
	RequiresJoinApproval bool
}

// SetSPOHubSite updates a hub site (get by URL → SetProperty → Update).
func (s *Service) SetSPOHubSite(ctx context.Context, p SetSPOHubSiteParams) (*spoapi.Result, error) {
	res, err := s.C.Invoke(ctx, spoapi.Op{
		CmdletName: "Set-SPOHubSite",
		TypeID:     spoapi.TenantTypeID,
		Chain:      hubChain(p.Identity),
		Set:        hubSetProps(p.Title, p.Description, p.LogoUrl, p.RequiresJoinApproval),
		Update:     true,
		Query:      true,
	})
	if err != nil {
		return nil, err
	}
	if o := res.First(); o != nil {
		o["Id"] = p.Identity
		o["Identity"] = p.Identity
	}
	return res, nil
}

// RemoveSPOHubSiteParams targets a hub site by its site URL.
type RemoveSPOHubSiteParams struct{ Identity string }

// RemoveSPOHubSite unregisters a hub site (Tenant.UnregisterHubSite(url)).
func (s *Service) RemoveSPOHubSite(ctx context.Context, p RemoveSPOHubSiteParams) (*spoapi.Result, error) {
	return s.C.Invoke(ctx, spoapi.Op{
		CmdletName: "Unregister-SPOHubSite",
		TypeID:     spoapi.TenantTypeID,
		Invoke:     []spoapi.MethodCall{{Name: "UnregisterHubSite", Params: []spoapi.Param{{Type: spoapi.TypeString, Value: p.Identity}}}},
	})
}
