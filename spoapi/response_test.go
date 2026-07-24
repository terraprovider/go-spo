package spoapi

import "testing"

func TestParseResponseSuccess(t *testing.T) {
	// A read (queryID=3): header, ObjectPath result (id 2), Query result (id 3).
	raw := []byte(`[
      {"SchemaVersion":"15.0.0.0","LibraryVersion":"16.0.0.0","ErrorInfo":null,"TraceCorrelationId":"c"},
      2,{"IsNull":false},
      3,{"_ObjectType_":"Microsoft.Online.SharePoint.TenantAdministration.Tenant","_ObjectIdentity_":"id","SharingCapability":1,"OneDriveStorageQuota":1048576,"AIBuilderEnabled":true}
    ]`)
	val, err := parseResponse(raw, 3)
	if err != nil {
		t.Fatalf("parseResponse: %v", err)
	}
	if len(val) != 1 {
		t.Fatalf("got %d objects, want 1", len(val))
	}
	obj := val[0]
	if obj["SharingCapability"].(float64) != 1 {
		t.Errorf("SharingCapability = %v", obj["SharingCapability"])
	}
	if obj["AIBuilderEnabled"] != true {
		t.Errorf("AIBuilderEnabled = %v", obj["AIBuilderEnabled"])
	}
}

func TestParseResponseError(t *testing.T) {
	raw := []byte(`[{"SchemaVersion":"15.0.0.0","LibraryVersion":"16.0.0.0","ErrorInfo":{"ErrorMessage":"Access denied.","ErrorValue":null,"TraceCorrelationId":"t","ErrorCode":-2147024891,"ErrorTypeName":"Microsoft.SharePoint.Client.ServerUnauthorizedAccessException"},"TraceCorrelationId":"t"}]`)
	_, err := parseResponse(raw, 3)
	if err == nil {
		t.Fatal("expected an error")
	}
	ae, ok := err.(*APIError)
	if !ok {
		t.Fatalf("want *APIError, got %T", err)
	}
	if ae.Message != "Access denied." {
		t.Errorf("Message = %q", ae.Message)
	}
	if ae.Code != "Microsoft.SharePoint.Client.ServerUnauthorizedAccessException" {
		t.Errorf("Code = %q", ae.Code)
	}
	if IsNotFound(err) {
		t.Error("access-denied should not be not-found")
	}
}

func TestParseResponseNotFound(t *testing.T) {
	raw := []byte(`[{"SchemaVersion":"15.0.0.0","LibraryVersion":"16.0.0.0","ErrorInfo":{"ErrorMessage":"Cannot get site https://x.sharepoint.com/sites/gone.","ErrorValue":null,"TraceCorrelationId":"t","ErrorCode":-1,"ErrorTypeName":"Microsoft.SharePoint.Client.ServerException"},"TraceCorrelationId":"t"}]`)
	_, err := parseResponse(raw, 3)
	if !IsNotFound(err) {
		t.Errorf("expected not-found, got %v", err)
	}
}

func TestParseResponseWriteOnly(t *testing.T) {
	// A write with no read-back (queryID=0): header only (Update returns nothing).
	raw := []byte(`[{"SchemaVersion":"15.0.0.0","LibraryVersion":"16.0.0.0","ErrorInfo":null,"TraceCorrelationId":"c"}]`)
	val, err := parseResponse(raw, 0)
	if err != nil {
		t.Fatalf("parseResponse: %v", err)
	}
	if val != nil {
		t.Errorf("write-only Value = %v, want nil", val)
	}
}
