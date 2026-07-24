package spo

import (
	"context"
	"fmt"
	"strings"

	"github.com/terraprovider/go-spo/spoapi"
)

const siteDesignRightsView = 1 // TenantSiteDesignPrincipalRights.View

// currentDesignRights reads the principals granted rights on a design via the static
// Tenant.GetSiteDesignRights(id) (returns TenantSiteDesignPrincipal objects).
func (s *Service) currentDesignRights(ctx context.Context, id string) ([]string, error) {
	items, err := s.C.InvokeStaticList(ctx, "Get-SPOSiteDesignRights", spoapi.TenantTypeID, "GetSiteDesignRights", []spoapi.Param{{Type: spoapi.TypeGuid, Value: id}})
	if err != nil {
		return nil, err
	}
	var out []string
	for _, it := range items {
		if n, _ := it["PrincipalName"].(string); n != "" {
			out = append(out, n)
		}
	}
	return out, nil
}

func (s *Service) grantDesignRights(ctx context.Context, id string, principals []string) error {
	if len(principals) == 0 {
		return nil
	}
	_, err := s.C.Invoke(ctx, spoapi.Op{
		CmdletName: "Grant-SPOSiteDesignRights",
		TypeID:     spoapi.TenantTypeID,
		Invoke: []spoapi.MethodCall{{Name: "GrantSiteDesignRights", Params: []spoapi.Param{
			{Type: spoapi.TypeGuid, Value: id},
			{Type: spoapi.TypeArray, ElemType: spoapi.TypeString, Value: principals},
			{Type: spoapi.TypeEnum, Value: siteDesignRightsView},
		}}},
	})
	return err
}

func (s *Service) revokeDesignRights(ctx context.Context, id string, principals []string) error {
	if len(principals) == 0 {
		return nil
	}
	// RevokeSiteDesignRights is a static method.
	return s.C.InvokeStaticVoid(ctx, "Revoke-SPOSiteDesignRights", spoapi.TenantTypeID, "RevokeSiteDesignRights", []spoapi.Param{
		{Type: spoapi.TypeGuid, Value: id},
		{Type: spoapi.TypeArray, ElemType: spoapi.TypeString, Value: principals},
	})
}

// NewSPOSiteDesignRightsParams grant View rights on a site design to principals.
type NewSPOSiteDesignRightsParams struct {
	Identity   string // the site design id (GUID)
	Principals []string
}

// NewSPOSiteDesignRights grants View rights to the principals.
func (s *Service) NewSPOSiteDesignRights(ctx context.Context, p NewSPOSiteDesignRightsParams) (*spoapi.Result, error) {
	return &spoapi.Result{}, s.grantDesignRights(ctx, p.Identity, p.Principals)
}

// GetSPOSiteDesignRightsParams targets the rights of a site design by id.
type GetSPOSiteDesignRightsParams struct{ Identity string }

// GetSPOSiteDesignRights reads the principals granted View rights on a design.
func (s *Service) GetSPOSiteDesignRights(ctx context.Context, p GetSPOSiteDesignRightsParams) (*spoapi.Result, error) {
	principals, err := s.currentDesignRights(ctx, p.Identity)
	if err != nil {
		return nil, err
	}
	return &spoapi.Result{Value: []map[string]any{{"Identity": p.Identity, "Id": p.Identity, "Principals": toAny(principals)}}}, nil
}

// SetSPOSiteDesignRightsParams reconcile the granted principals.
type SetSPOSiteDesignRightsParams struct {
	Identity   string
	Principals []string
}

// SetSPOSiteDesignRights reconciles View rights (grants new, revokes gone).
func (s *Service) SetSPOSiteDesignRights(ctx context.Context, p SetSPOSiteDesignRightsParams) (*spoapi.Result, error) {
	cur, err := s.currentDesignRights(ctx, p.Identity)
	if err != nil {
		return nil, err
	}
	want, have := map[string]bool{}, map[string]bool{}
	for _, n := range p.Principals {
		want[strings.ToLower(n)] = true
	}
	for _, n := range cur {
		have[strings.ToLower(n)] = true
	}
	var grant, revoke []string
	for _, n := range p.Principals {
		if !have[strings.ToLower(n)] {
			grant = append(grant, n)
		}
	}
	for _, n := range cur {
		if !want[strings.ToLower(n)] {
			revoke = append(revoke, n)
		}
	}
	if err := s.grantDesignRights(ctx, p.Identity, grant); err != nil {
		return nil, fmt.Errorf("grant: %w", err)
	}
	if err := s.revokeDesignRights(ctx, p.Identity, revoke); err != nil {
		return nil, fmt.Errorf("revoke: %w", err)
	}
	return &spoapi.Result{}, nil
}

// RemoveSPOSiteDesignRightsParams targets the rights of a site design by id.
type RemoveSPOSiteDesignRightsParams struct{ Identity string }

// RemoveSPOSiteDesignRights revokes all granted rights.
func (s *Service) RemoveSPOSiteDesignRights(ctx context.Context, p RemoveSPOSiteDesignRightsParams) (*spoapi.Result, error) {
	cur, err := s.currentDesignRights(ctx, p.Identity)
	if err != nil {
		return nil, err
	}
	return &spoapi.Result{}, s.revokeDesignRights(ctx, p.Identity, cur)
}
