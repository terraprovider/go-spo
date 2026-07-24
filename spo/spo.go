// Package spo is the typed binding surface over the SharePoint admin CSOM
// transport (spoapi). The Get/Set bindings per admin object (tenant singleton,
// site collections) are generated from the spo-powershell-api-re catalog into
// zz_generated_spo.go; this file holds the stable, hand-written pieces.
package spo

//go:generate go run ../cmd/gen-go -out zz_generated_spo.go

import (
	"github.com/terraprovider/go-spo/spoapi"
)

// Service exposes the typed SharePoint admin operations over a spoapi.Client.
type Service struct {
	C *spoapi.Client
}

// New wraps a spoapi.Client.
func New(c *spoapi.Client) *Service { return &Service{C: c} }
