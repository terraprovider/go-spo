package spo

import (
	"context"
	"strings"

	"github.com/terraprovider/go-spo/spoapi"
)

var restrictedSearchModes = map[string]int{"Disabled": 0, "Enabled": 1}
var restrictedSearchModeNames = map[int]string{0: "Disabled", 1: "Enabled"}

// GetSPORestrictedSearchParams reads the tenant restricted-search config (singleton).
type GetSPORestrictedSearchParams struct{}

// GetSPORestrictedSearch reads the restricted-search mode and allowed site list.
func (s *Service) GetSPORestrictedSearch(ctx context.Context, _ GetSPORestrictedSearchParams) (*spoapi.Result, error) {
	mode, err := s.C.MethodInt(ctx, "Get-SPORestrictedSearchMode", spoapi.TenantTypeID, "GetSPORestrictedSearchMode", nil)
	if err != nil {
		return nil, err
	}
	list, err := s.C.MethodStrings(ctx, "Get-SPORestrictedSearchAllowedList", spoapi.TenantTypeID, "GetSPORestrictedSearchAllowedList", nil)
	if err != nil {
		return nil, err
	}
	obj := map[string]any{
		"Identity":    s.C.AdminURL(),
		"Id":          s.C.AdminURL(),
		"Mode":        restrictedSearchModeNames[mode],
		"AllowedList": toAny(list),
	}
	return &spoapi.Result{Value: []map[string]any{obj}}, nil
}

// SetSPORestrictedSearchParams configure restricted search.
type SetSPORestrictedSearchParams struct {
	Mode        string // "Disabled" | "Enabled"
	AllowedList []string
}

// SetSPORestrictedSearch sets the mode and reconciles the allowed site list.
func (s *Service) SetSPORestrictedSearch(ctx context.Context, p SetSPORestrictedSearchParams) (*spoapi.Result, error) {
	cur, err := s.C.MethodStrings(ctx, "Get-SPORestrictedSearchAllowedList", spoapi.TenantTypeID, "GetSPORestrictedSearchAllowedList", nil)
	if err != nil {
		return nil, err
	}
	want := map[string]bool{}
	for _, u := range p.AllowedList {
		want[strings.ToLower(u)] = true
	}
	have := map[string]bool{}
	for _, u := range cur {
		have[strings.ToLower(u)] = true
	}
	var toAdd, toRemove []string
	for _, u := range p.AllowedList {
		if !have[strings.ToLower(u)] {
			toAdd = append(toAdd, u)
		}
	}
	for _, u := range cur {
		if !want[strings.ToLower(u)] {
			toRemove = append(toRemove, u)
		}
	}
	invoke := []spoapi.MethodCall{{
		Name:   "SetSPORestrictedSearchMode",
		Params: []spoapi.Param{{Type: spoapi.TypeEnum, Value: restrictedSearchModes[p.Mode]}},
	}}
	if len(toAdd) > 0 {
		invoke = append(invoke, spoapi.MethodCall{Name: "AddSPORestrictedSearchAllowedList", Params: []spoapi.Param{{Type: spoapi.TypeArray, ElemType: spoapi.TypeString, Value: toAdd}}})
	}
	if len(toRemove) > 0 {
		invoke = append(invoke, spoapi.MethodCall{Name: "RemoveSPORestrictedSearchAllowedList", Params: []spoapi.Param{{Type: spoapi.TypeArray, ElemType: spoapi.TypeString, Value: toRemove}}})
	}
	return s.C.Invoke(ctx, spoapi.Op{CmdletName: "Set-SPORestrictedSearch", TypeID: spoapi.TenantTypeID, Invoke: invoke})
}
