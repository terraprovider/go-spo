package spo

import (
	"context"

	"github.com/terraprovider/go-spo/spoapi"
)

// GetSPOHomeSiteParams reads the tenant home site (singleton).
type GetSPOHomeSiteParams struct{}

// GetSPOHomeSite reads the primary tenant home site URL (Tenant.GetSPHSiteUrl()).
func (s *Service) GetSPOHomeSite(ctx context.Context, _ GetSPOHomeSiteParams) (*spoapi.Result, error) {
	url, err := s.C.MethodString(ctx, "Get-SPOHomeSite", spoapi.TenantTypeID, "GetSPHSiteUrl", nil)
	if err != nil {
		return nil, err
	}
	return &spoapi.Result{Value: []map[string]any{{"Identity": s.C.AdminURL(), "Id": s.C.AdminURL(), "Url": url}}}, nil
}

// SetSPOHomeSiteParams set the tenant home site.
type SetSPOHomeSiteParams struct{ Url string }

// SetSPOHomeSite sets the tenant home site (Tenant.AddHomeSite(url, 0, [])).
func (s *Service) SetSPOHomeSite(ctx context.Context, p SetSPOHomeSiteParams) (*spoapi.Result, error) {
	return s.C.Invoke(ctx, spoapi.Op{
		CmdletName: "Set-SPOHomeSite",
		TypeID:     spoapi.TenantTypeID,
		Invoke: []spoapi.MethodCall{{Name: "AddHomeSite", Params: []spoapi.Param{
			{Type: spoapi.TypeString, Value: p.Url},
			{Type: spoapi.TypeInt32, Value: 0},
			{Type: spoapi.TypeArray, ElemType: spoapi.TypeGuid, Value: []string{}},
		}}},
	})
}
