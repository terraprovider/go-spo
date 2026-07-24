package spoapi

import (
	"os"
	"path/filepath"
	"testing"
)

// The fixtures in testdata/processquery are emitted by the shipped CSOM runtime
// (spo-powershell-api-re/tools/emit-processquery-samples.ps1). buildRequest must
// reproduce them byte-for-byte, proving the Go transport's wire is
// indistinguishable from the module's.
func TestBuildRequestGolden(t *testing.T) {
	cases := []struct {
		file string
		op   Op
	}{
		{"read-tenant.xml", Op{TypeID: TenantTypeID, Query: true}},
		{"set-boolean.xml", Op{TypeID: TenantTypeID, Update: true, Set: []SetProp{
			{Name: "AIBuilderEnabled", Type: TypeBoolean, Value: true}}}},
		{"set-string.xml", Op{TypeID: TenantTypeID, Update: true, Set: []SetProp{
			{Name: "AIBuilderDefaultPowerAppsEnvironment", Type: TypeString, Value: "env-abc123"}}}},
		{"set-int32.xml", Op{TypeID: TenantTypeID, Update: true, Set: []SetProp{
			{Name: "EmailAttestationReAuthDays", Type: TypeInt32, Value: 30}}}},
		{"set-int64.xml", Op{TypeID: TenantTypeID, Update: true, Set: []SetProp{
			{Name: "OneDriveStorageQuota", Type: TypeInt64, Value: int64(1048576)}}}},
		{"set-enum.xml", Op{TypeID: TenantTypeID, Update: true, Set: []SetProp{
			{Name: "SharingCapability", Type: TypeEnum, Value: 1}}}}, // ExternalUserSharingOnly
		{"set-guid.xml", Op{TypeID: TenantTypeID, Update: true, Set: []SetProp{
			{Name: "AllOrganizationSecurityGroupId", Type: TypeGuid, Value: "11111111-2222-3333-4444-555555555555"}}}},
		{"set-array-guid.xml", Op{TypeID: TenantTypeID, Update: true, Set: []SetProp{
			{Name: "AllowedDomainListForSyncClient", Type: TypeArray, ElemType: TypeGuid, Value: []string{
				"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "ffffffff-0000-1111-2222-333333333333"}}}}},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			want, err := os.ReadFile(filepath.Join("testdata", "processquery", tc.file))
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			got, _, err := buildRequest(tc.op, ".NET Library")
			if err != nil {
				t.Fatalf("buildRequest: %v", err)
			}
			if got != string(want) {
				t.Errorf("wire mismatch for %s\n got: %s\nwant: %s", tc.file, got, want)
			}
		})
	}
}

// The generalized builder must also reproduce the object-path shapes (method
// paths, method actions, collection queries) byte-for-byte.
func TestBuildRequestObjectPathsGolden(t *testing.T) {
	const siteURL = "https://contoso.sharepoint.com/sites/marketing"
	cases := []struct {
		file string
		op   Op
	}{
		// Tenant.GetSitePropertiesByUrl(url, true) + query-all.
		{"op-site-get.xml", Op{TypeID: TenantTypeID, Query: true, Chain: []PathStep{
			{Method: "GetSitePropertiesByUrl", Params: []Param{
				{Type: TypeString, Value: siteURL}, {Type: TypeBoolean, Value: true}}}}}},
		// Tenant.GetSiteProperties(0, true) + query-all collection.
		{"op-site-list.xml", Op{TypeID: TenantTypeID, Query: true, QueryChildItems: true, Chain: []PathStep{
			{Method: "GetSiteProperties", Params: []Param{
				{Type: TypeInt32, Value: 0}, {Type: TypeBoolean, Value: true}}}}}},
		// Tenant.RemoveSite(url) — method object path, no query.
		{"op-site-remove.xml", Op{TypeID: TenantTypeID, Chain: []PathStep{
			{Method: "RemoveSite", Params: []Param{{Type: TypeString, Value: siteURL}}}}}},
		// Tenant.AddTenantTheme(name, json) — terminal method action with params.
		{"op-theme-add.xml", Op{TypeID: TenantTypeID, Invoke: []MethodCall{
			{Name: "AddTenantTheme", Params: []Param{
				{Type: TypeString, Value: "MyTheme"}, {Type: TypeString, Value: `{"palette":{}}`}}}}}},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			want, err := os.ReadFile(filepath.Join("testdata", "processquery", tc.file))
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			got, _, err := buildRequest(tc.op, ".NET Library")
			if err != nil {
				t.Fatalf("buildRequest: %v", err)
			}
			if got != string(want) {
				t.Errorf("wire mismatch for %s\n got: %s\nwant: %s", tc.file, got, want)
			}
		})
	}
}

// The <Query> action's Id is the response key for the read result.
func TestBuildRequestQueryID(t *testing.T) {
	// read-only: ObjectPath=2, Query=3
	if _, id, _ := buildRequest(Op{TypeID: TenantTypeID, Query: true}, ".NET Library"); id != 3 {
		t.Errorf("read queryID = %d, want 3", id)
	}
	// write + read-back: ObjectPath=2, Set=3, Update=4, Query=5
	op := Op{TypeID: TenantTypeID, Update: true, Query: true, Set: []SetProp{{Name: "X", Type: TypeBoolean, Value: true}}}
	if _, id, _ := buildRequest(op, ".NET Library"); id != 5 {
		t.Errorf("write+read queryID = %d, want 5", id)
	}
	// write-only: no query
	if _, id, _ := buildRequest(Op{TypeID: TenantTypeID, Update: true, Set: []SetProp{{Name: "X", Type: TypeBoolean, Value: true}}}, ".NET Library"); id != 0 {
		t.Errorf("write-only queryID = %d, want 0", id)
	}
}

func TestTextEscape(t *testing.T) {
	if got := textEscape(`a & b < c > d`); got != `a &amp; b &lt; c &gt; d` {
		t.Errorf("textEscape = %q", got)
	}
}
