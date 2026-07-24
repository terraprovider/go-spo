package spec

import "testing"

func TestTenantCatalogLoads(t *testing.T) {
	c, err := Tenant()
	if err != nil {
		t.Fatal(err)
	}
	if c.Object != "Microsoft.Online.SharePoint.TenantAdministration.Tenant" {
		t.Errorf("Object = %q", c.Object)
	}
	if len(c.Properties) < 300 {
		t.Errorf("only %d properties", len(c.Properties))
	}
	if n := len(c.Knobs()); n < 150 {
		t.Errorf("only %d knobs, expected the full Set-SPOTenant surface", n)
	}
	// SharingCapability is a well-known enum knob; sanity-check the catalog shape.
	var found bool
	for _, p := range c.Knobs() {
		if p.Name == "SharingCapability" {
			found = true
			if !p.IsEnum || p.EnumType == "" {
				t.Errorf("SharingCapability should be an enum knob: %+v", p)
			}
			if _, ok := c.Enums[p.EnumType]; !ok {
				t.Errorf("SharingCapability enum %q missing from catalog", p.EnumType)
			}
		}
	}
	if !found {
		t.Error("SharingCapability knob not found")
	}
}

func TestCloudsLoad(t *testing.T) {
	e, err := Clouds()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := e.Environments["Commercial"]; !ok {
		t.Error("Commercial cloud missing")
	}
}
