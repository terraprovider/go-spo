package spo

import (
	"context"
	"errors"
	"fmt"

	"github.com/terraprovider/go-spo/spoapi"
)

// NewSPOThemeParams are the create inputs (Add-SPOTheme). Identity is the theme name.
type NewSPOThemeParams struct {
	Identity  string
	ThemeJson string
}

// addTheme builds AddTenantTheme(name, themeJson) — create and update both overwrite.
func (s *Service) addTheme(ctx context.Context, cmdlet, name, json string) (*spoapi.Result, error) {
	return s.C.Invoke(ctx, spoapi.Op{
		CmdletName: cmdlet,
		TypeID:     spoapi.TenantTypeID,
		Invoke: []spoapi.MethodCall{{
			Name: "AddTenantTheme",
			Params: []spoapi.Param{
				{Type: spoapi.TypeString, Value: name},
				{Type: spoapi.TypeString, Value: json},
			},
		}},
	})
}

// NewSPOTheme adds a tenant theme (Tenant.AddTenantTheme(name, json)).
func (s *Service) NewSPOTheme(ctx context.Context, p NewSPOThemeParams) (*spoapi.Result, error) {
	return s.addTheme(ctx, "Add-SPOTheme", p.Identity, p.ThemeJson)
}

// GetSPOThemeParams targets a theme by name.
type GetSPOThemeParams struct{ Identity string }

// GetSPOTheme reads a theme by name (Tenant.GetTenantTheme(name)).
func (s *Service) GetSPOTheme(ctx context.Context, p GetSPOThemeParams) (*spoapi.Result, error) {
	res, err := s.C.Invoke(ctx, spoapi.Op{
		CmdletName: "Get-SPOTheme",
		TypeID:     spoapi.TenantTypeID,
		Chain:      []spoapi.PathStep{{Method: "GetTenantTheme", Params: []spoapi.Param{{Type: spoapi.TypeString, Value: p.Identity}}}},
		Query:      true,
	})
	if err != nil {
		// GetTenantTheme signals a missing theme with a generic 400 "Unknown Error"
		// rather than a null result; map that to not-found.
		var ae *spoapi.APIError
		if errors.As(err, &ae) && ae.Status == 400 {
			return nil, &spoapi.APIError{Status: 404, Message: fmt.Sprintf("theme %q not found", p.Identity)}
		}
		return nil, err
	}
	o := res.First()
	// GetTenantTheme returns an object with a null Name (not a server error) for a
	// missing theme.
	if o == nil || isNull(o) || o["Name"] == nil || fmt.Sprint(o["Name"]) == "" {
		return nil, &spoapi.APIError{Status: 404, Message: fmt.Sprintf("theme %q not found", p.Identity)}
	}
	o["Id"] = p.Identity
	o["Identity"] = p.Identity
	return res, nil
}

// SetSPOThemeParams are the update inputs (theme name + JSON; Add overwrites).
type SetSPOThemeParams struct {
	Identity  string
	ThemeJson string
}

// SetSPOTheme overwrites a theme. AddTenantTheme refuses to overwrite an existing
// theme, so update = delete + re-add (same name) in one ProcessQuery.
func (s *Service) SetSPOTheme(ctx context.Context, p SetSPOThemeParams) (*spoapi.Result, error) {
	return s.C.Invoke(ctx, spoapi.Op{
		CmdletName: "Set-SPOTheme",
		TypeID:     spoapi.TenantTypeID,
		Invoke: []spoapi.MethodCall{
			{Name: "DeleteTenantTheme", Params: []spoapi.Param{{Type: spoapi.TypeString, Value: p.Identity}}},
			{Name: "AddTenantTheme", Params: []spoapi.Param{{Type: spoapi.TypeString, Value: p.Identity}, {Type: spoapi.TypeString, Value: p.ThemeJson}}},
		},
	})
}

// RemoveSPOThemeParams targets a theme by name.
type RemoveSPOThemeParams struct{ Identity string }

// RemoveSPOTheme deletes a theme (Tenant.DeleteTenantTheme(name)).
func (s *Service) RemoveSPOTheme(ctx context.Context, p RemoveSPOThemeParams) (*spoapi.Result, error) {
	return s.C.Invoke(ctx, spoapi.Op{
		CmdletName: "Remove-SPOTheme",
		TypeID:     spoapi.TenantTypeID,
		Invoke:     []spoapi.MethodCall{{Name: "DeleteTenantTheme", Params: []spoapi.Param{{Type: spoapi.TypeString, Value: p.Identity}}}},
	})
}

// isNull reports whether a CSOM object result is a null server object.
func isNull(o map[string]any) bool {
	b, _ := o["ServerObjectIsNull"].(bool)
	return b
}
