package spoapi

import "testing"

// buildStaticUpdate must reproduce the runtime's Set-SPOSiteScript wire byte-for-byte:
// construct Tenant → static GetSiteScript(id) → SetProperty(s) → UpdateSiteScript(by ref).
// Captured from the shipped CSOM runtime (spo-powershell-api-re decompilation showed
// Set-SPOSiteScript calls Tenant.GetSiteScript then UpdateSiteScript).
func TestBuildStaticUpdateGolden(t *testing.T) {
	u := StaticUpdate{
		RootTypeID: TenantTypeID,
		GetMethod:  "GetSiteScript",
		GetParams:  []Param{{Type: TypeGuid, Value: "11111111-1111-1111-1111-111111111111"}},
		Set: []SetProp{
			{Name: "Title", Type: TypeString, Value: "NewTitle"},
			{Name: "Content", Type: TypeString, Value: "{}"},
		},
		UpdateMethod: "UpdateSiteScript",
	}
	want := `<Request AddExpandoFieldTypeSuffix="true" SchemaVersion="15.0.0.0" LibraryVersion="16.0.0.0" ApplicationName=".NET Library" xmlns="http://schemas.microsoft.com/sharepoint/clientquery/2009"><Actions><ObjectPath Id="2" ObjectPathId="1" /><ObjectPath Id="4" ObjectPathId="3" /><SetProperty Id="5" ObjectPathId="3" Name="Title"><Parameter Type="String">NewTitle</Parameter></SetProperty><SetProperty Id="6" ObjectPathId="3" Name="Content"><Parameter Type="String">{}</Parameter></SetProperty><ObjectPath Id="8" ObjectPathId="7" /></Actions><ObjectPaths><Constructor Id="1" TypeId="{268004ae-ef6b-4e9b-8425-127220d84719}" /><StaticMethod Id="3" Name="GetSiteScript" TypeId="{268004ae-ef6b-4e9b-8425-127220d84719}"><Parameters><Parameter Type="Guid">{11111111-1111-1111-1111-111111111111}</Parameter></Parameters></StaticMethod><Method Id="7" ParentId="1" Name="UpdateSiteScript"><Parameters><Parameter ObjectPathId="3" /></Parameters></Method></ObjectPaths></Request>`
	got, err := buildStaticUpdate(u, ".NET Library")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("static-update wire mismatch\n got: %s\nwant: %s", got, want)
	}
}
