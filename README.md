# go-snap-bi

A Go implementation of Bank Indonesia's **SNAP** (Standar Nasional Open API
Pembayaran) payment standard, document version **1.0.2** (September 2024),
covering every API Service category published on the
[ASPI SNAP Developer Site](https://apidevportal.aspi-indonesia.or.id/api-services).

```
go get github.com/koriebruh/go-snap-bi
```

## Status

All 7 of the portal's API Service categories are accounted for — **79
typed endpoint bindings** across 5 domain packages, plus the shared
signing/token/transport core:

| Portal category      | Package                                                    | Service Codes  | Endpoints |
|-----------------------|------------------------------------------------------------|-----------------|-----------|
| Registrasi            | [`registration`](./registration)                            | 01–10, 81       | 11        |
| Informasi Saldo       | [`balanceinfo`](./balanceinfo)                               | 11              | 1         |
| Riwayat Transaksi     | [`transactionhistory`](./transactionhistory)                 | 12–14           | 3         |
| Transfer Kredit       | [`transfercredit`](./transfercredit)                         | 15–53, 75–78    | 43        |
| Transfer Debit        | [`transferdebit`](./transferdebit)                           | 54–72, 79–80    | 21        |
| Keamanan              | root `snap` package (`token.go`)                             | 73–74           | —         |
| Administrasi          | *(no API endpoints on the portal — onboarding docs only)*    | —               | —         |

Keamanan's Access Token B2B/B2B2C endpoints live in the root package
because every other package depends on them to get a token in the first
place — they're authentication infrastructure, not a peer domain
endpoint.

See [`doc.go`](./doc.go) (`go doc github.com/koriebruh/go-snap-bi`) for
the full package-layout writeup, including where to add a new endpoint
when the portal changes.

## Design

- **One core, five domain packages.** The root `snap` package holds only
  what every domain package needs: request signing (HMAC/RSA), the
  access-token lifecycle, server-side inbound-request verification,
  header assembly, response-code parsing, and the shared `Money` type.
  Each domain package maps 1:1 to an ASPI portal category, so "which
  package does this endpoint belong in" is never a judgment call.
- **One shape per endpoint.** Every calling function follows the same
  pattern: marshal the typed request, sign and send it, check the
  response status (HTTP status is authoritative over the SNAP
  `responseCode` body, never the other way around), and unmarshal into
  a typed response.
- **Wire-shape fidelity over convenience.** Mandatory/Optional/Conditional
  field markers from the standard are preserved exactly as `omitempty`
  presence; genuinely ambiguous or unspecified-shape fields are modeled
  as `json.RawMessage` rather than guessed at.
- **Zero third-party dependencies.** Everything is Go standard library.

## Usage

### Signing a request (symmetric / HMAC)

```go
hb := snap.HeaderBuilder{
	Method:       http.MethodPost,
	EndpointURL:  "https://partner.example.com/v1.0/balance-inquiry",
	AccessToken:  accessToken, // from TokenManager.AccessTokenB2B, below
	ClientKey:    clientKey,
	PartnerID:    partnerID,
	ExternalID:   externalID,
	ChannelID:    channelID,
	Symmetric:    true,
	ClientSecret: clientSecret,
}

transport := &snap.Transport{}
resp, err := balanceinfo.BalanceInquiry(ctx, transport, hb, balanceinfo.BalanceInquiryRequest{
	PartnerReferenceNo: "2020102900000000000001",
	AccountNo:          "1234567890",
})
if err != nil {
	// errors.Is(err, snap.ErrBadRequest), snap.ErrUnauthorized, etc. —
	// one sentinel per SNAP response-code HTTP-status class.
}
```

Every domain package follows the identical call shape:
`func Endpoint(ctx, *snap.Transport, snap.HeaderBuilder, Request) (Response, error)`.

### Getting an access token (B2B)

```go
tm := &snap.TokenManager{
	BaseURL:   "https://partner.example.com",
	ClientKey: clientKey,
	Signer:    rsaPrivateKey, // crypto.Signer
}
token, err := tm.AccessTokenB2B(ctx)
```

`TokenManager` caches and refreshes the token automatically; concurrent
callers during a refresh share one in-flight request rather than each
issuing their own.

### Verifying an inbound (server-side) request

For endpoints where this package is the receiver — payment
notifications, callbacks — use `snap.ServerVerifier` with a `KeyStore`
implementation to validate the signature on an incoming request before
trusting its body.

## Requirements

- Go 1.21 or later (see [`go.mod`](./go.mod)).

## Contributing

Endpoint bindings are added one Service Code at a time, each following
the conventions documented in `doc.go` and the per-phase design notes
under [`docs/superpowers/specs/`](./docs/superpowers/specs/). Field
tables and worked examples are sourced from the ASPI SNAP Developer
Site; research notes recording the portal's own contradictions and
ambiguities live under [`docs/research/`](./docs/research/).

## Author

[JamalKya Nanami](https://github.com/koriebruh) ([@koriebruh](https://github.com/koriebruh))

## License

Not yet licensed — a `LICENSE` file has not been added to this
repository yet. Until one is added, no license is granted for use,
copying, or redistribution beyond what's permitted by default copyright
law.
