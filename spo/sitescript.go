package spo

import (
	"context"
	"fmt"
	"strings"

	"github.com/terraprovider/go-spo/spoapi"
)

// CSOM server type ids (ScriptTypeAttribute.ServerTypeId) for the site-script
// object parameters, read from the shipped runtime.
const (
	siteScriptCreationInfoTypeID = "{7cce1194-93c4-44a2-9a2a-92094fd345e5}" // TenantSiteScriptCreationInfo
	tenantSiteScriptTypeID       = "{717c203d-a629-47df-80bb-cdeda6592aa4}" // TenantSiteScript
)

// NewSPOSiteScriptParams are the create inputs (Add-SPOSiteScript).
type NewSPOSiteScriptParams struct {
	Title       string
	Description string
	Content     string
}

// NewSPOSiteScript creates a site script (Tenant.CreateSiteScript(info)) and returns
// the created object with its server-assigned Id (exposed as Identity).
func (s *Service) NewSPOSiteScript(ctx context.Context, p NewSPOSiteScriptParams) (*spoapi.Result, error) {
	res, err := s.C.Invoke(ctx, spoapi.Op{
		CmdletName: "Add-SPOSiteScript",
		TypeID:     spoapi.TenantTypeID,
		Chain: []spoapi.PathStep{{
			Method: "CreateSiteScript",
			Params: []spoapi.Param{{
				ObjectTypeID: siteScriptCreationInfoTypeID,
				Props: []spoapi.SetProp{
					{Name: "Content", Type: spoapi.TypeString, Value: p.Content},
					{Name: "ContentStream", Type: spoapi.TypeNull},
					{Name: "Description", Type: spoapi.TypeString, Value: p.Description},
					{Name: "Title", Type: spoapi.TypeString, Value: p.Title},
				},
			}},
		}},
		Query: true,
	})
	if err != nil {
		return nil, err
	}
	normalizeSiteScript(res.First())
	return res, nil
}

// GetSPOSiteScriptParams targets a site script by Id.
type GetSPOSiteScriptParams struct{ Identity string }

// GetSPOSiteScript reads one site script by Id (Tenant.GetSiteScripts() filtered).
func (s *Service) GetSPOSiteScript(ctx context.Context, p GetSPOSiteScriptParams) (*spoapi.Result, error) {
	res, err := s.C.Invoke(ctx, spoapi.Op{
		CmdletName:      "Get-SPOSiteScript",
		TypeID:          spoapi.TenantTypeID,
		Chain:           []spoapi.PathStep{{Method: "GetSiteScripts"}},
		Query:           true,
		QueryChildItems: true,
	})
	if err != nil {
		return nil, err
	}
	want := strings.ToLower(strings.Trim(p.Identity, "{}"))
	for _, item := range res.ChildItems() {
		if strings.ToLower(spoapi.DecodeGuid(item["Id"])) == want {
			normalizeSiteScript(item)
			return &spoapi.Result{Value: []map[string]any{item}}, nil
		}
	}
	return nil, &spoapi.APIError{Status: 404, Message: fmt.Sprintf("site script %q not found", p.Identity)}
}

// SetSPOSiteScriptParams are the update inputs (Set-SPOSiteScript -Identity <id>).
type SetSPOSiteScriptParams struct {
	Identity    string // the site-script Id
	Title       string
	Description string
	Content     string
}

// SetSPOSiteScript updates a site script. Mirrors Set-SPOSiteScript: resolve the
// object via the static Tenant.GetSiteScript(id), set the changed properties, and
// pass it by reference to Tenant.UpdateSiteScript. (A TenantSiteScript is a CSOM
// ClientObject and cannot be sent as an inline object parameter.)
func (s *Service) SetSPOSiteScript(ctx context.Context, p SetSPOSiteScriptParams) (*spoapi.Result, error) {
	return s.C.InvokeStaticUpdate(ctx, spoapi.StaticUpdate{
		CmdletName: "Set-SPOSiteScript",
		RootTypeID: spoapi.TenantTypeID,
		GetMethod:  "GetSiteScript",
		GetParams:  []spoapi.Param{{Type: spoapi.TypeGuid, Value: p.Identity}},
		Set: []spoapi.SetProp{
			{Name: "Title", Type: spoapi.TypeString, Value: p.Title},
			{Name: "Content", Type: spoapi.TypeString, Value: p.Content},
			{Name: "Description", Type: spoapi.TypeString, Value: p.Description},
		},
		UpdateMethod: "UpdateSiteScript",
	})
}

// RemoveSPOSiteScriptParams targets a site script by Id.
type RemoveSPOSiteScriptParams struct{ Identity string }

// RemoveSPOSiteScript deletes a site script (Tenant.DeleteSiteScript(id)).
func (s *Service) RemoveSPOSiteScript(ctx context.Context, p RemoveSPOSiteScriptParams) (*spoapi.Result, error) {
	return s.C.Invoke(ctx, spoapi.Op{
		CmdletName: "Remove-SPOSiteScript",
		TypeID:     spoapi.TenantTypeID,
		Invoke: []spoapi.MethodCall{{
			Name:   "DeleteSiteScript",
			Params: []spoapi.Param{{Type: spoapi.TypeGuid, Value: p.Identity}},
		}},
	})
}

// normalizeSiteScript decodes the CSOM Id (/Guid(…)/) to a bare GUID and exposes it
// as Identity so the resource has a stable key.
func normalizeSiteScript(obj map[string]any) {
	if obj == nil {
		return
	}
	if id := spoapi.DecodeGuid(obj["Id"]); id != "" {
		obj["Id"] = id
		obj["Identity"] = id
	}
}
