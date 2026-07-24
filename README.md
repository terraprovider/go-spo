# go-spo

A Go client for the **SharePoint Online tenant/site admin** API — the **CSOM**
(Client-Side Object Model) surface behind the `Microsoft.Online.SharePoint.PowerShell`
module (`Connect-SPOService` / the `-SPO*` cmdlets). Typed bindings are **generated**
from a derived catalog so the surface tracks the module; the transport reproduces the
module's exact ProcessQuery wire so traffic is indistinguishable from the PowerShell
client.

> Not affiliated with or endorsed by Microsoft. It calls the same documented admin
> CSOM API the PowerShell module does. Built on
> [`go-msadmin`](https://github.com/terraprovider/go-msadmin); the Terraform provider
> builds on [`tf-msadmin`](https://github.com/terraprovider/tf-msadmin).

```bash
go get github.com/terraprovider/go-spo
```

## Status

- **`spoapi`** (CSOM transport core) — ✅ implemented, unit-tested against
  byte-exact golden fixtures emitted by the shipped CSOM runtime. Live validation
  pending a tenant.
- **`spec`** (embedded derived catalog: 356 Tenant properties, 185 knobs, enums) — ✅.
- **`cmd/gen-go`** → **`spo`** (typed bindings) — ✅ for the **tenant-settings
  singleton** (`Get-SPOTenant` / `Set-SPOTenant`, 184 typed knobs). Site collections
  next.

```go
svc := spo.New(client)
res, _ := svc.GetSPOTenant(ctx)          // read the tenant admin object
_, _  = svc.SetSPOTenant(ctx, spo.SetSPOTenantParams{
    SharingCapability:    ptr("ExternalUserSharingOnly"), // enum by name
    OneDriveStorageQuota: ptr(int64(1048576)),
})
```

## `spoapi` — the transport

CSOM is not REST: the client batches actions against object paths as XML and POSTs
them to **one** endpoint.

```go
c, _ := spoapi.New(spoapi.Options{
    Environment: spoapi.Commercial,       // or GCCHigh / DoD / China
    TenantName:  "contoso",               // → https://contoso-admin.sharepoint.com
    Tokens:      tokenProvider,           // any TokenProvider (see Auth)
})

// Read the tenant object (construct + query-all):
res, _ := c.Invoke(ctx, spoapi.Op{CmdletName: "Get-SPOTenant", TypeID: spoapi.TenantTypeID, Query: true})
obj := res.First() // map[string]any of the Tenant properties
```

It reproduces the module faithfully (see `spo-powershell-api-re/docs/02-csom-protocol.md`):

- **ProcessQuery** — `POST /_vti_bin/client.svc/ProcessQuery` with a CSOM `<Request>`
  (`<ObjectPaths>` + `<Actions>`); the tenant admin object is a `<Constructor>` of
  type `{268004ae-…}`. Reads use `<Query SelectAllProperties="true">`; writes use
  `<SetProperty>` per knob + `<Method Name="Update">`.
- **Typed parameter serialisation** — `Boolean`/`String`/`Int32`/`Int64`/`Enum`/
  `Guid`/`Array`, matching the catalog's per-property wire type.
- **Errors inside 200** — SharePoint returns `ErrorInfo` in an HTTP 200; `spoapi`
  surfaces it as `httpx.APIError`, with `spoapi.IsNotFound(err)`.
- `br`/`gzip`/`deflate` response decoding; throttling/`Retry-After`/5xx retries via
  `go-msadmin/retry` (override `Options.HTTPClient`).

## Auth

`spoapi.TokenProvider` is `Token(ctx, resource) (string, error)`; the **resource is the
tenant admin URL** (`https://{tenant}-admin.sharepoint.com`). App-only needs an Entra
app with SharePoint **`Sites.FullControl.All`**. An `authx.Config.Build()` or a
`msalauth` provider (go-exoscc) satisfies it.

> ⚠️ **Client-secret app-only auth is not supported by SharePoint.** The CSOM
> endpoint rejects secret-based app tokens with `401 Unsupported app only token`
> (it requires `appidacr=2`, i.e. a **certificate** credential) — this is refused
> *before* any permission check, so the certificate is mandatory, not optional.
> Use `ARM_CLIENT_CERTIFICATE_PATH` (PEM or PKCS#12) or an equivalent.
> Confirmed live against a real tenant.

## Generation

The private `spo-powershell-api-re` factory reflects the module → `spec/catalog/*.json`
(+ golden ProcessQuery fixtures); `spec` embeds it; `cmd/gen-go` emits
`spo/zz_generated_spo.go`. Regenerate with `go generate ./spo/`. Only the derived
factual catalog is committed here — never Microsoft binaries or decompiled sources.

## License

[MIT](LICENSE) © glueckanja AG
