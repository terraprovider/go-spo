package spo

import (
	"context"

	"github.com/terraprovider/go-spo/spoapi"
)

const listDesignCreationInfoTypeID = "{4039ef3a-3ca7-4ce2-8164-e52b1215bc79}"

// NewSPOListDesignParams are the create inputs (Add-SPOListDesign). List designs
// have no update method; changing an attribute recreates the design. SiteScripts
// is mandatory server-side (Add-SPOListDesign requires at least one).
type NewSPOListDesignParams struct {
	Title       string
	Description string
	SiteScripts []string
}

// NewSPOListDesign creates a list design (Tenant.CreateListDesign(info)).
func (s *Service) NewSPOListDesign(ctx context.Context, p NewSPOListDesignParams) (*spoapi.Result, error) {
	// SiteScriptIds is mandatory (a null array yields "Unknown Error"); always send
	// the array form, even if empty.
	scripts := spoapi.SetProp{Name: "SiteScriptIds", Type: spoapi.TypeArray, ElemType: spoapi.TypeGuid, Value: p.SiteScripts}
	res, err := s.C.Invoke(ctx, spoapi.Op{
		CmdletName: "Add-SPOListDesign", TypeID: spoapi.TenantTypeID,
		Chain: []spoapi.PathStep{{Method: "CreateListDesign", Params: []spoapi.Param{{
			ObjectTypeID: listDesignCreationInfoTypeID,
			Props: []spoapi.SetProp{
				optStr("Description", p.Description),
				{Name: "ListColor", Type: spoapi.TypeEnum, Value: 0},
				{Name: "ListIcon", Type: spoapi.TypeEnum, Value: 0},
				scripts,
				{Name: "TemplateFeatures", Type: spoapi.TypeNull},
				{Name: "ThumbnailUrl", Type: spoapi.TypeNull},
				{Name: "Title", Type: spoapi.TypeString, Value: p.Title},
			},
		}}}},
		Query: true,
	})
	if err != nil {
		return nil, err
	}
	normalizeSiteDesign(res.First()) // same Id/SiteScriptIds shape
	return res, nil
}

// GetSPOListDesignParams targets a list design by Id.
type GetSPOListDesignParams struct{ Identity string }

// GetSPOListDesign reads one list design (static Tenant.GetListDesign(id)).
func (s *Service) GetSPOListDesign(ctx context.Context, p GetSPOListDesignParams) (*spoapi.Result, error) {
	res, err := s.C.InvokeStaticGet(ctx, "Get-SPOListDesign", spoapi.TenantTypeID, "GetListDesign", []spoapi.Param{{Type: spoapi.TypeGuid, Value: p.Identity}})
	if err != nil {
		return nil, err
	}
	o := res.First()
	if o == nil {
		return nil, &spoapi.APIError{Status: 404, Message: "list design " + p.Identity + " not found"}
	}
	normalizeSiteDesign(o)
	return res, nil
}

// SetSPOListDesignParams is unused at runtime: list designs have no update method,
// so every attribute forces replacement. It exists only so the generated resource's
// Update path compiles; it is never invoked (a change recreates the design).
type SetSPOListDesignParams struct {
	Identity    string
	Title       string
	Description string
	SiteScripts []string
}

// SetSPOListDesign is a never-called stub (list designs are immutable — see
// SetSPOListDesignParams). It returns an error should it ever be reached.
func (s *Service) SetSPOListDesign(ctx context.Context, p SetSPOListDesignParams) (*spoapi.Result, error) {
	return nil, &spoapi.APIError{Status: 400, Message: "list designs cannot be updated; a change forces replacement"}
}

// RemoveSPOListDesignParams targets a list design by Id.
type RemoveSPOListDesignParams struct{ Identity string }

// RemoveSPOListDesign deletes a list design (Tenant.RemoveListDesign(id)).
func (s *Service) RemoveSPOListDesign(ctx context.Context, p RemoveSPOListDesignParams) (*spoapi.Result, error) {
	return s.C.Invoke(ctx, spoapi.Op{
		CmdletName: "Remove-SPOListDesign", TypeID: spoapi.TenantTypeID,
		Invoke: []spoapi.MethodCall{{Name: "RemoveListDesign", Params: []spoapi.Param{{Type: spoapi.TypeGuid, Value: p.Identity}}}},
	})
}
