// Package spec embeds the derived SharePoint admin catalog published from the
// private spo-powershell-api-re factory (reflection of the SPO Management Shell +
// its CSOM object model). It is the single source of truth for code generation:
// cmd/gen-go turns it into typed bindings, and the Terraform provider's cmd/gen-tf
// turns it into resources. Only factual interface metadata is committed — no
// Microsoft sources.
package spec

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed catalog/tenant-catalog.json
var tenantCatalogJSON []byte

//go:embed catalog/site-catalog.json
var siteCatalogJSON []byte

//go:embed environments.json
var environmentsJSON []byte

// ObjectCatalog is the descriptor for one CSOM admin object mapped to a resource
// (the Tenant singleton, a SiteProperties site collection, …): every property, its
// ProcessQuery wire type, and whether the Set-SPO* cmdlet exposes it as a knob.
type ObjectCatalog struct {
	Source         string                    `json:"source"`
	Object         string                    `json:"object"`
	GetCmdlet      string                    `json:"getCmdlet"`
	SetCmdlet      string                    `json:"setCmdlet"`
	KeyParam       string                    `json:"keyParam"`
	ReadableCount  int                       `json:"readableCount"`
	SettableCount  int                       `json:"settableCount"`
	PropertyCount  int                       `json:"propertyCount"`
	Properties     []Property                `json:"properties"`
	Enums          map[string]map[string]int `json:"enums"`
	OrphanSetParam []string                  `json:"orphanSetParams"`
}

// TenantCatalog is retained for back-compat; every object uses ObjectCatalog.
type TenantCatalog = ObjectCatalog

// Property is one CSOM property of an admin object.
type Property struct {
	Name         string   `json:"name"`
	CsomType     string   `json:"csomType"` // Boolean|String|Int32|Int64|Enum|Guid|Array|…
	IsEnum       bool     `json:"isEnum"`
	EnumType     string   `json:"enumType"` // full .NET enum type name (key into Enums)
	IsCollection bool     `json:"isCollection"`
	ElementType  string   `json:"elementType"` // Array element wire type
	Readable     bool     `json:"readable"`
	Settable     bool     `json:"settable"`
	ViaSetCmdlet bool     `json:"viaSetCmdlet"`
	IsKnob       bool     `json:"isKnob"` // settable AND surfaced by Set-SPOTenant (1:1)
	ValidateSet  []string `json:"validateSet"`
}

// Environments is the per-cloud endpoint table.
type Environments struct {
	Source           string                 `json:"source"`
	ProductionClouds []string               `json:"productionClouds"`
	Environments     map[string]Environment `json:"environments"`
}

// Environment is one national cloud's SharePoint/AAD endpoints.
type Environment struct {
	Region            string `json:"region"`
	SPOSuffix         string `json:"spoSuffix"`
	AdminHostTemplate string `json:"adminHostTemplate"`
	Authority         string `json:"authority"`
	GraphHost         string `json:"graphHost"`
	Verified          bool   `json:"verified"`
}

// Tenant parses and returns the embedded tenant-settings catalog.
func Tenant() (*ObjectCatalog, error) { return parseCatalog(tenantCatalogJSON, "tenant-catalog.json") }

// Site parses and returns the embedded site-collection catalog.
func Site() (*ObjectCatalog, error) { return parseCatalog(siteCatalogJSON, "site-catalog.json") }

func parseCatalog(data []byte, name string) (*ObjectCatalog, error) {
	var c ObjectCatalog
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("spec: parse %s: %w", name, err)
	}
	return &c, nil
}

// Clouds parses and returns the embedded environments table.
func Clouds() (*Environments, error) {
	var e Environments
	if err := json.Unmarshal(environmentsJSON, &e); err != nil {
		return nil, fmt.Errorf("spec: parse environments.json: %w", err)
	}
	return &e, nil
}

// Knobs returns the tenant properties Set-SPOTenant exposes as admin knobs.
func (c *TenantCatalog) Knobs() []Property {
	var out []Property
	for _, p := range c.Properties {
		if p.IsKnob {
			out = append(out, p)
		}
	}
	return out
}
