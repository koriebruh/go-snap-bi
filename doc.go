// Package snap implements the shared security and transport envelope for
// Bank Indonesia's SNAP (Standar Nasional Open API Pembayaran) standard,
// document version 1.0.2 (September 2024).
//
// Phase 1 scope only: signing primitives, token lifecycle, server-side
// verification, header assembly with per-PJP quirk hooks, and response-code
// parsing. Per-service request/response bindings (balance inquiry, transfer,
// virtual account, QRIS, direct debit, etc.) are added incrementally in
// later phases. See docs/superpowers/specs/2026-09-10-go-snap-bi-core-design.md.
package snap
