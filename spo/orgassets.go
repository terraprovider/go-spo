package spo

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/terraprovider/go-spo/spoapi"
)

var orgAssetTypes = map[string]int{
	"ImageDocumentLibrary": 1, "OfficeTemplateLibrary": 2, "OfficeFontLibrary": 4,
}
var orgAssetTypeNames = map[int]string{1: "ImageDocumentLibrary", 2: "OfficeTemplateLibrary", 4: "OfficeFontLibrary"}

// urlParam renders a string method-parameter, or a Null parameter when empty (the
// API rejects "" for optional URL args like thumbnailUrl with "Invalid ThumbnailUrl").
func urlParam(v string) spoapi.Param {
	if v == "" {
		return spoapi.Param{Type: spoapi.TypeNull}
	}
	return spoapi.Param{Type: spoapi.TypeString, Value: v}
}

// resourcePathURL extracts the DecodedUrl from an SP.ResourcePath value (or a bare
// string). GetOrgAssets returns LibraryUrl as {DecodedUrl: "<server-relative>"}.
func resourcePathURL(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case map[string]any:
		if s, _ := t["DecodedUrl"].(string); s != "" {
			return s
		}
	}
	return ""
}

func normURL(u string) string { return strings.ToLower(strings.TrimRight(u, "/")) }

// orgAssetLibrary finds the org-assets library entry for libUrl (via GetOrgAssets()).
// GetOrgAssets returns each LibraryUrl as a *server-relative* path (e.g. "Shared
// Documents") under a Domain (the site collection URL); reconstruct the absolute
// URL from Domain + LibraryUrl to match the caller's full library URL.
func (s *Service) orgAssetLibrary(ctx context.Context, libUrl string) (map[string]any, error) {
	rm, err := s.C.InvokeMethodResult(ctx, "Get-SPOOrgAssetsLibrary", spoapi.TenantTypeID, "GetOrgAssets", nil)
	if err != nil {
		return nil, err
	}
	var oa struct {
		Domain map[string]any `json:"Domain"`
		Libs   struct {
			Child []map[string]any `json:"_Child_Items_"`
		} `json:"OrgAssetsLibraries"`
	}
	if rm != nil {
		_ = json.Unmarshal(rm, &oa)
	}
	domain := strings.TrimRight(resourcePathURL(oa.Domain), "/")
	want := normURL(libUrl)
	for _, lib := range oa.Libs.Child {
		rel := resourcePathURL(lib["LibraryUrl"])
		if rel == "" {
			continue
		}
		full := rel
		if !strings.HasPrefix(strings.ToLower(rel), "http") && domain != "" {
			full = domain + "/" + strings.TrimLeft(rel, "/")
		}
		// match on the absolute URL, or (fallback) on the server-relative tail
		if normURL(full) == want || strings.HasSuffix(want, "/"+normURL(rel)) {
			return lib, nil
		}
	}
	return nil, nil
}

// NewSPOOrgAssetsLibraryParams add an org-assets library (keyed by library URL).
type NewSPOOrgAssetsLibraryParams struct {
	Identity     string // the document library URL
	ThumbnailUrl string
	OrgAssetType string // ImageDocumentLibrary | OfficeTemplateLibrary | OfficeFontLibrary
	CdnType      string // Public | Private (default Private)
}

// NewSPOOrgAssetsLibrary adds a library to org assets (AddToOrgAssetsLibAndCdnWithType).
func (s *Service) NewSPOOrgAssetsLibrary(ctx context.Context, p NewSPOOrgAssetsLibraryParams) (*spoapi.Result, error) {
	cdn := cdnTypes[p.CdnType]
	if p.CdnType == "" {
		cdn = 1 // Private
	}
	return s.C.Invoke(ctx, spoapi.Op{
		CmdletName: "Add-SPOOrgAssetsLibrary", TypeID: spoapi.TenantTypeID,
		Invoke: []spoapi.MethodCall{{Name: "AddToOrgAssetsLibAndCdnWithType", Params: []spoapi.Param{
			{Type: spoapi.TypeEnum, Value: cdn},
			{Type: spoapi.TypeString, Value: p.Identity},
			urlParam(p.ThumbnailUrl),
			{Type: spoapi.TypeEnum, Value: orgAssetTypes[p.OrgAssetType]},
		}}},
	})
}

// GetSPOOrgAssetsLibraryParams targets an org-assets library by URL.
type GetSPOOrgAssetsLibraryParams struct{ Identity string }

// GetSPOOrgAssetsLibrary reads an org-assets library entry.
func (s *Service) GetSPOOrgAssetsLibrary(ctx context.Context, p GetSPOOrgAssetsLibraryParams) (*spoapi.Result, error) {
	lib, err := s.orgAssetLibrary(ctx, p.Identity)
	if err != nil {
		return nil, err
	}
	if lib == nil {
		return nil, &spoapi.APIError{Status: 404, Message: "org assets library " + p.Identity + " not found"}
	}
	obj := map[string]any{"Identity": p.Identity, "Id": p.Identity}
	if f, ok := lib["OrgAssetType"].(float64); ok {
		obj["OrgAssetType"] = orgAssetTypeNames[int(f)]
	}
	return &spoapi.Result{Value: []map[string]any{obj}}, nil
}

// SetSPOOrgAssetsLibraryParams update an org-assets library.
type SetSPOOrgAssetsLibraryParams struct {
	Identity     string
	ThumbnailUrl string
	OrgAssetType string
}

// SetSPOOrgAssetsLibrary updates an org-assets library (SetOrgAssetsWithType).
func (s *Service) SetSPOOrgAssetsLibrary(ctx context.Context, p SetSPOOrgAssetsLibraryParams) (*spoapi.Result, error) {
	return s.C.Invoke(ctx, spoapi.Op{
		CmdletName: "Set-SPOOrgAssetsLibrary", TypeID: spoapi.TenantTypeID,
		Invoke: []spoapi.MethodCall{{Name: "SetOrgAssetsWithType", Params: []spoapi.Param{
			{Type: spoapi.TypeString, Value: p.Identity},
			urlParam(p.ThumbnailUrl),
			{Type: spoapi.TypeEnum, Value: orgAssetTypes[p.OrgAssetType]},
		}}},
	})
}

// RemoveSPOOrgAssetsLibraryParams targets an org-assets library by URL.
type RemoveSPOOrgAssetsLibraryParams struct{ Identity string }

const emptyGuid = "00000000-0000-0000-0000-000000000000"

// RemoveSPOOrgAssetsLibrary removes a library from org assets by URL. The API
// (RemoveFromOrgAssets) accepts only one of LibraryUrl/ListId — pass the URL and
// an empty GUID for the ListId (mirroring Remove-SPOOrgAssetsLibrary -LibraryUrl).
func (s *Service) RemoveSPOOrgAssetsLibrary(ctx context.Context, p RemoveSPOOrgAssetsLibraryParams) (*spoapi.Result, error) {
	return s.C.Invoke(ctx, spoapi.Op{
		CmdletName: "Remove-SPOOrgAssetsLibrary", TypeID: spoapi.TenantTypeID,
		Invoke: []spoapi.MethodCall{{Name: "RemoveFromOrgAssets", Params: []spoapi.Param{
			{Type: spoapi.TypeString, Value: p.Identity},
			{Type: spoapi.TypeGuid, Value: emptyGuid},
		}}},
	})
}
