# Security Policy

## Scope

This project implements cryptographic request signing (HMAC-SHA512
symmetric, SHA256withRSA/PKCS#1v1.5 asymmetric), OAuth2 access-token
lifecycle management, and inbound-request signature verification for
Bank Indonesia SNAP payment integrations. It handles PJP credentials
(client keys/secrets, RSA private keys) and payment-transaction data
that callers pass in; it does not itself store or transmit them beyond
one HTTP request/response cycle.

This is an independent, community-maintained implementation. It is not
affiliated with, endorsed by, or officially certified by Bank
Indonesia or ASPI.

## Supported Versions

Only the latest tagged release is supported. This project is pre-1.0
(see the [README](./README.md)'s versioning note) — no long-term
support branches exist yet.

## Reporting a Vulnerability

Please do not open a public GitHub issue for a security
vulnerability. Instead, report it privately via [GitHub's private
vulnerability reporting](https://github.com/koriebruh/go-snap-bi/security/advisories/new)
on this repository, or by contacting the maintainer
([@koriebruh](https://github.com/koriebruh)) directly.

Include what you found, how to reproduce it, and its potential impact.
Expect an initial response within a reasonable timeframe from a
single-maintainer open-source project; there is no formal SLA.

## Security Checks

Every push and pull request runs, via CI (`.github/workflows/ci.yml`):

- `go build` / `go vet` / `gofmt`
- `go test -race`
- `golangci-lint`
- `govulncheck` (known-vulnerability scan against the Go standard
  library and any dependencies)
- `gosec` (static security analysis)

A release tag additionally re-runs the full build/vet/test gate before
a GitHub Release is published (`.github/workflows/release.yml`) — see
[CONFORMANCE.md](./CONFORMANCE.md) for what "passing" covers and does
not cover.
