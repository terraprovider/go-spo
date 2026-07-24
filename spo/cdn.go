package spo

import (
	"context"
	"fmt"
	"strings"

	"github.com/terraprovider/go-spo/spoapi"
)

// SPOTenantCdnType values.
var cdnTypes = map[string]int{"Public": 0, "Private": 1}

func cdnTypeValue(s string) (int, error) {
	if v, ok := cdnTypes[s]; ok {
		return v, nil
	}
	return 0, fmt.Errorf("spo: cdn type must be Public or Private, got %q", s)
}

func toAny(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

// GetSPOTenantCdnParams targets the Office 365 CDN config for one type.
type GetSPOTenantCdnParams struct{ Identity string } // "Public" | "Private"

// GetSPOTenantCdn reads whether the CDN of a type is enabled and its origins.
func (s *Service) GetSPOTenantCdn(ctx context.Context, p GetSPOTenantCdnParams) (*spoapi.Result, error) {
	ct, err := cdnTypeValue(p.Identity)
	if err != nil {
		return nil, err
	}
	enabled, err := s.C.MethodBool(ctx, "Get-SPOTenantCdnEnabled", spoapi.TenantTypeID, "GetTenantCdnEnabled", []spoapi.Param{{Type: spoapi.TypeEnum, Value: ct}})
	if err != nil {
		return nil, err
	}
	origins, err := s.C.MethodStrings(ctx, "Get-SPOTenantCdnOrigins", spoapi.TenantTypeID, "GetTenantCdnOrigins", []spoapi.Param{{Type: spoapi.TypeEnum, Value: ct}})
	if err != nil {
		return nil, err
	}
	obj := map[string]any{"Identity": p.Identity, "Id": p.Identity, "Enabled": enabled, "Origins": toAny(origins)}
	return &spoapi.Result{Value: []map[string]any{obj}}, nil
}

// SetSPOTenantCdnParams configure the CDN of a type.
type SetSPOTenantCdnParams struct {
	Identity string
	Enabled  bool
	Origins  []string
}

// SetSPOTenantCdn enables/disables the CDN of a type and reconciles its origins
// (adds new, removes gone) in a single ProcessQuery.
func (s *Service) SetSPOTenantCdn(ctx context.Context, p SetSPOTenantCdnParams) (*spoapi.Result, error) {
	ct, err := cdnTypeValue(p.Identity)
	if err != nil {
		return nil, err
	}
	cur, err := s.C.MethodStrings(ctx, "Get-SPOTenantCdnOrigins", spoapi.TenantTypeID, "GetTenantCdnOrigins", []spoapi.Param{{Type: spoapi.TypeEnum, Value: ct}})
	if err != nil {
		return nil, err
	}
	want := map[string]bool{}
	for _, o := range p.Origins {
		want[strings.ToLower(o)] = true
	}
	have := map[string]bool{}
	for _, o := range cur {
		have[strings.ToLower(o)] = true
	}
	invoke := []spoapi.MethodCall{{Name: "SetTenantCdnEnabled", Params: []spoapi.Param{{Type: spoapi.TypeEnum, Value: ct}, {Type: spoapi.TypeBoolean, Value: p.Enabled}}}}
	for _, o := range p.Origins {
		if !have[strings.ToLower(o)] {
			invoke = append(invoke, spoapi.MethodCall{Name: "AddTenantCdnOrigin", Params: []spoapi.Param{{Type: spoapi.TypeEnum, Value: ct}, {Type: spoapi.TypeString, Value: o}}})
		}
	}
	for _, o := range cur {
		if !want[strings.ToLower(o)] {
			invoke = append(invoke, spoapi.MethodCall{Name: "RemoveTenantCdnOrigin", Params: []spoapi.Param{{Type: spoapi.TypeEnum, Value: ct}, {Type: spoapi.TypeString, Value: o}}})
		}
	}
	return s.C.Invoke(ctx, spoapi.Op{CmdletName: "Set-SPOTenantCdn", TypeID: spoapi.TenantTypeID, Invoke: invoke})
}
