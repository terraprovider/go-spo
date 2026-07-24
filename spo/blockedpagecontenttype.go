package spo

import (
	"context"
	"encoding/json"

	"github.com/terraprovider/go-spo/spoapi"
)

// TemplateFileType values for blocked page-creation content types.
var templateFileTypes = map[string]int{"StandardPage": 0, "WikiPage": 1, "FormPage": 2, "ClientSidePage": 3}

// NewSPOBlockedPageContentTypeParams blocks creation of a page content type (keyed by name).
type NewSPOBlockedPageContentTypeParams struct{ Identity string }

// NewSPOBlockedPageContentType blocks a content type (Tenant.AddBlockedPageCreationContentType).
func (s *Service) NewSPOBlockedPageContentType(ctx context.Context, p NewSPOBlockedPageContentTypeParams) (*spoapi.Result, error) {
	return s.C.Invoke(ctx, spoapi.Op{
		CmdletName: "Set-SPOBlockedPageCreationContentType",
		TypeID:     spoapi.TenantTypeID,
		Invoke:     []spoapi.MethodCall{{Name: "AddBlockedPageCreationContentType", Params: []spoapi.Param{{Type: spoapi.TypeEnum, Value: templateFileTypes[p.Identity]}}}},
	})
}

// GetSPOBlockedPageContentTypeParams targets a blocked content type by name.
type GetSPOBlockedPageContentTypeParams struct{ Identity string }

// GetSPOBlockedPageContentType reports whether a content type is blocked.
func (s *Service) GetSPOBlockedPageContentType(ctx context.Context, p GetSPOBlockedPageContentTypeParams) (*spoapi.Result, error) {
	rm, err := s.C.InvokeMethodResult(ctx, "Get-SPOBlockedPageCreationContentTypeList", spoapi.TenantTypeID, "GetBlockedPageCreationContentTypes", nil)
	if err != nil {
		return nil, err
	}
	var vals []int
	_ = json.Unmarshal(rm, &vals)
	want := templateFileTypes[p.Identity]
	for _, v := range vals {
		if v == want {
			return &spoapi.Result{Value: []map[string]any{{"Identity": p.Identity, "Id": p.Identity}}}, nil
		}
	}
	return nil, &spoapi.APIError{Status: 404, Message: "blocked content type " + p.Identity + " not found"}
}

// RemoveSPOBlockedPageContentTypeParams targets a blocked content type by name.
type RemoveSPOBlockedPageContentTypeParams struct{ Identity string }

// RemoveSPOBlockedPageContentType unblocks a content type (Tenant.RemoveBlockedPageCreationContentType).
func (s *Service) RemoveSPOBlockedPageContentType(ctx context.Context, p RemoveSPOBlockedPageContentTypeParams) (*spoapi.Result, error) {
	return s.C.Invoke(ctx, spoapi.Op{
		CmdletName: "Remove-SPOBlockedPageCreationContentType",
		TypeID:     spoapi.TenantTypeID,
		Invoke:     []spoapi.MethodCall{{Name: "RemoveBlockedPageCreationContentType", Params: []spoapi.Param{{Type: spoapi.TypeEnum, Value: templateFileTypes[p.Identity]}}}},
	})
}
