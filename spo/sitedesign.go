package spo

import (
	"context"
	"strings"

	"github.com/terraprovider/go-spo/spoapi"
)

// TenantSiteDesignCreationInfo server type id (from the CreateSiteDesign wire).
const siteDesignCreationInfoTypeID = "{52bd24b0-327a-4fb4-8323-783645e48cf0}"

// NewSPOSiteDesignParams are the create inputs (Add-SPOSiteDesign).
type NewSPOSiteDesignParams struct {
	Title               string
	WebTemplate         string   // "64" (team site) | "68" (communication site) | "1" (group)
	SiteScripts         []string // ordered site-script GUIDs
	Description         string
	PreviewImageUrl     string
	PreviewImageAltText string
	IsDefault           bool
}

func optStr(name, v string) spoapi.SetProp {
	if v == "" {
		return spoapi.SetProp{Name: name, Type: spoapi.TypeNull}
	}
	return spoapi.SetProp{Name: name, Type: spoapi.TypeString, Value: v}
}

// NewSPOSiteDesign creates a site design (Tenant.CreateSiteDesign(info)).
func (s *Service) NewSPOSiteDesign(ctx context.Context, p NewSPOSiteDesignParams) (*spoapi.Result, error) {
	scripts := spoapi.SetProp{Name: "SiteScriptIds", Type: spoapi.TypeNull}
	if len(p.SiteScripts) > 0 {
		scripts = spoapi.SetProp{Name: "SiteScriptIds", Type: spoapi.TypeArray, ElemType: spoapi.TypeGuid, Value: p.SiteScripts}
	}
	res, err := s.C.Invoke(ctx, spoapi.Op{
		CmdletName: "Add-SPOSiteDesign",
		TypeID:     spoapi.TenantTypeID,
		Chain: []spoapi.PathStep{{
			Method: "CreateSiteDesign",
			Params: []spoapi.Param{{
				ObjectTypeID: siteDesignCreationInfoTypeID,
				Props: []spoapi.SetProp{ // alphabetical, matching the runtime
					optStr("Description", p.Description),
					{Name: "DesignPackageId", Type: spoapi.TypeGuid, Value: "00000000-0000-0000-0000-000000000000"},
					{Name: "IsDefault", Type: spoapi.TypeBoolean, Value: p.IsDefault},
					optStr("PreviewImageAltText", p.PreviewImageAltText),
					optStr("PreviewImageUrl", p.PreviewImageUrl),
					scripts,
					{Name: "ThumbnailUrl", Type: spoapi.TypeNull},
					{Name: "Title", Type: spoapi.TypeString, Value: p.Title},
					{Name: "WebTemplate", Type: spoapi.TypeString, Value: p.WebTemplate},
				},
			}},
		}},
		Query: true,
	})
	if err != nil {
		return nil, err
	}
	normalizeSiteDesign(res.First())
	return res, nil
}

// GetSPOSiteDesignParams targets a site design by Id.
type GetSPOSiteDesignParams struct{ Identity string }

// GetSPOSiteDesign reads one site design (static Tenant.GetSiteDesign(id)).
func (s *Service) GetSPOSiteDesign(ctx context.Context, p GetSPOSiteDesignParams) (*spoapi.Result, error) {
	res, err := s.C.InvokeStaticGet(ctx, "Get-SPOSiteDesign", spoapi.TenantTypeID, "GetSiteDesign",
		[]spoapi.Param{{Type: spoapi.TypeGuid, Value: p.Identity}})
	if err != nil {
		return nil, err
	}
	if o := res.First(); o != nil {
		normalizeSiteDesign(o)
	} else {
		return nil, &spoapi.APIError{Status: 404, Message: "site design " + p.Identity + " not found"}
	}
	return res, nil
}

// SetSPOSiteDesignParams are the update inputs (Set-SPOSiteDesign -Identity <id>).
type SetSPOSiteDesignParams struct {
	Identity            string
	Title               string
	WebTemplate         string
	SiteScripts         []string
	Description         string
	PreviewImageUrl     string
	PreviewImageAltText string
	IsDefault           bool
}

// SetSPOSiteDesign updates a site design (static GetSiteDesign → SetProperty →
// UpdateSiteDesign by reference).
func (s *Service) SetSPOSiteDesign(ctx context.Context, p SetSPOSiteDesignParams) (*spoapi.Result, error) {
	set := []spoapi.SetProp{
		{Name: "Title", Type: spoapi.TypeString, Value: p.Title},
		{Name: "WebTemplate", Type: spoapi.TypeString, Value: p.WebTemplate},
		{Name: "Description", Type: spoapi.TypeString, Value: p.Description},
		{Name: "PreviewImageUrl", Type: spoapi.TypeString, Value: p.PreviewImageUrl},
		{Name: "PreviewImageAltText", Type: spoapi.TypeString, Value: p.PreviewImageAltText},
		{Name: "IsDefault", Type: spoapi.TypeBoolean, Value: p.IsDefault},
	}
	if len(p.SiteScripts) > 0 {
		set = append(set, spoapi.SetProp{Name: "SiteScriptIds", Type: spoapi.TypeArray, ElemType: spoapi.TypeGuid, Value: p.SiteScripts})
	}
	return s.C.InvokeStaticUpdate(ctx, spoapi.StaticUpdate{
		CmdletName:   "Set-SPOSiteDesign",
		RootTypeID:   spoapi.TenantTypeID,
		GetMethod:    "GetSiteDesign",
		GetParams:    []spoapi.Param{{Type: spoapi.TypeGuid, Value: p.Identity}},
		Set:          set,
		UpdateMethod: "UpdateSiteDesign",
	})
}

// RemoveSPOSiteDesignParams targets a site design by Id.
type RemoveSPOSiteDesignParams struct{ Identity string }

// RemoveSPOSiteDesign deletes a site design (Tenant.DeleteSiteDesign(id)).
func (s *Service) RemoveSPOSiteDesign(ctx context.Context, p RemoveSPOSiteDesignParams) (*spoapi.Result, error) {
	return s.C.Invoke(ctx, spoapi.Op{
		CmdletName: "Remove-SPOSiteDesign",
		TypeID:     spoapi.TenantTypeID,
		Invoke: []spoapi.MethodCall{{
			Name:   "DeleteSiteDesign",
			Params: []spoapi.Param{{Type: spoapi.TypeGuid, Value: p.Identity}},
		}},
	})
}

// normalizeSiteDesign decodes the CSOM Id + SiteScriptIds (/Guid(…)/) and exposes a
// stable Identity.
func normalizeSiteDesign(obj map[string]any) {
	if obj == nil {
		return
	}
	if id := spoapi.DecodeGuid(obj["Id"]); id != "" {
		obj["Id"] = id
		obj["Identity"] = id
	}
	if raw, ok := obj["SiteScriptIds"].([]any); ok {
		out := make([]any, 0, len(raw))
		for _, g := range raw {
			out = append(out, strings.ToLower(spoapi.DecodeGuid(g)))
		}
		obj["SiteScriptIds"] = out
	}
}
