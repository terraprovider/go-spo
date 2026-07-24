package spo

import (
	"context"

	"github.com/terraprovider/go-spo/spoapi"
)

// GetSPOKnowledgeHubSiteParams reads the tenant knowledge hub site (singleton).
type GetSPOKnowledgeHubSiteParams struct{}

// GetSPOKnowledgeHubSite reads the tenant knowledge hub site URL (Tenant.GetKnowledgeHubSite()).
func (s *Service) GetSPOKnowledgeHubSite(ctx context.Context, _ GetSPOKnowledgeHubSiteParams) (*spoapi.Result, error) {
	url, err := s.C.MethodString(ctx, "Get-SPOKnowledgeHubSite", spoapi.TenantTypeID, "GetKnowledgeHubSite", nil)
	if err != nil {
		return nil, err
	}
	return &spoapi.Result{Value: []map[string]any{{"Identity": s.C.AdminURL(), "Id": s.C.AdminURL(), "Url": url}}}, nil
}

// SetSPOKnowledgeHubSiteParams set the tenant knowledge hub site.
type SetSPOKnowledgeHubSiteParams struct{ Url string }

// SetSPOKnowledgeHubSite sets the tenant knowledge hub site (Tenant.SetKnowledgeHubSite(url)).
func (s *Service) SetSPOKnowledgeHubSite(ctx context.Context, p SetSPOKnowledgeHubSiteParams) (*spoapi.Result, error) {
	return s.C.Invoke(ctx, spoapi.Op{
		CmdletName: "Set-SPOKnowledgeHubSite",
		TypeID:     spoapi.TenantTypeID,
		Invoke:     []spoapi.MethodCall{{Name: "SetKnowledgeHubSite", Params: []spoapi.Param{{Type: spoapi.TypeString, Value: p.Url}}}},
	})
}
