package spo

import (
	"context"
	"strings"

	"github.com/terraprovider/go-spo/spoapi"
)

// storage entities are keyed by "<site-url>|<key>" (the key on a site's root web,
// typically the tenant app catalog).
func splitStorage(id string) (siteURL, key string) {
	if i := strings.LastIndex(id, "|"); i >= 0 {
		return id[:i], id[i+1:]
	}
	return "", id
}

func siteWebChain(siteURL string) []spoapi.PathStep {
	return []spoapi.PathStep{
		{Method: "GetSiteByUrl", Params: []spoapi.Param{{Type: spoapi.TypeString, Value: siteURL}}},
		{Method: "RootWeb", Property: true},
	}
}

// NewSPOStorageEntityParams set a tenant storage entity ("<site-url>|<key>").
type NewSPOStorageEntityParams struct {
	Identity    string
	Value       string
	Description string
	Comment     string
}

func (s *Service) setStorage(ctx context.Context, cmdlet, id, value, desc, comment string) (*spoapi.Result, error) {
	siteURL, key := splitStorage(id)
	return s.C.Invoke(ctx, spoapi.Op{
		CmdletName: cmdlet, TypeID: spoapi.TenantTypeID, Chain: siteWebChain(siteURL),
		Invoke: []spoapi.MethodCall{{Name: "SetStorageEntity", Params: []spoapi.Param{
			{Type: spoapi.TypeString, Value: key}, {Type: spoapi.TypeString, Value: value},
			{Type: spoapi.TypeString, Value: desc}, {Type: spoapi.TypeString, Value: comment},
		}}},
	})
}

// NewSPOStorageEntity sets a storage entity (site.RootWeb.SetStorageEntity).
func (s *Service) NewSPOStorageEntity(ctx context.Context, p NewSPOStorageEntityParams) (*spoapi.Result, error) {
	return s.setStorage(ctx, "Set-SPOStorageEntity", p.Identity, p.Value, p.Description, p.Comment)
}

// GetSPOStorageEntityParams targets a storage entity ("<site-url>|<key>").
type GetSPOStorageEntityParams struct{ Identity string }

// GetSPOStorageEntity reads a storage entity (site.RootWeb.GetStorageEntity(key)).
func (s *Service) GetSPOStorageEntity(ctx context.Context, p GetSPOStorageEntityParams) (*spoapi.Result, error) {
	siteURL, key := splitStorage(p.Identity)
	chain := append(siteWebChain(siteURL), spoapi.PathStep{Method: "GetStorageEntity", Params: []spoapi.Param{{Type: spoapi.TypeString, Value: key}}})
	res, err := s.C.Invoke(ctx, spoapi.Op{CmdletName: "Get-SPOStorageEntity", TypeID: spoapi.TenantTypeID, Chain: chain, Query: true})
	if err != nil {
		return nil, err
	}
	o := res.First()
	if o == nil || isNull(o) {
		return nil, &spoapi.APIError{Status: 404, Message: "storage entity " + p.Identity + " not found"}
	}
	o["Identity"] = p.Identity
	o["Id"] = p.Identity
	return res, nil
}

// SetSPOStorageEntityParams update a storage entity.
type SetSPOStorageEntityParams struct {
	Identity    string
	Value       string
	Description string
	Comment     string
}

// SetSPOStorageEntity overwrites a storage entity.
func (s *Service) SetSPOStorageEntity(ctx context.Context, p SetSPOStorageEntityParams) (*spoapi.Result, error) {
	return s.setStorage(ctx, "Set-SPOStorageEntity", p.Identity, p.Value, p.Description, p.Comment)
}

// RemoveSPOStorageEntityParams targets a storage entity.
type RemoveSPOStorageEntityParams struct{ Identity string }

// RemoveSPOStorageEntity removes a storage entity (site.RootWeb.RemoveStorageEntity(key)).
func (s *Service) RemoveSPOStorageEntity(ctx context.Context, p RemoveSPOStorageEntityParams) (*spoapi.Result, error) {
	siteURL, key := splitStorage(p.Identity)
	return s.C.Invoke(ctx, spoapi.Op{
		CmdletName: "Remove-SPOStorageEntity", TypeID: spoapi.TenantTypeID, Chain: siteWebChain(siteURL),
		Invoke: []spoapi.MethodCall{{Name: "RemoveStorageEntity", Params: []spoapi.Param{{Type: spoapi.TypeString, Value: key}}}},
	})
}
