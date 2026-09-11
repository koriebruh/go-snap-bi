// Package snap implements Bank Indonesia's SNAP (Standar Nasional Open API
// Pembayaran) standard, document version 1.0.2 (September 2024).
//
// Phase 1 (complete): signing primitives, token lifecycle, server-side
// verification, header assembly with per-PJP quirk hooks, and response-code
// parsing — see docs/superpowers/specs/2026-09-10-go-snap-bi-core-design.md.
//
// Phase 2 (in progress): typed per-service request/response bindings built
// on Phase 1, added incrementally, one service at a time — see
// docs/superpowers/specs/2026-09-11-phase2-balance-inquiry-design.md for the
// first one (BalanceInquiry). Most services (transfer, virtual account,
// QRIS, direct debit, etc.) are not yet implemented.
package snap
