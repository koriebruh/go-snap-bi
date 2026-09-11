// Package snap implements Bank Indonesia's SNAP (Standar Nasional Open API
// Pembayaran) standard, document version 1.0.2 (September 2024).
//
// Phase 1 (complete): signing primitives, token lifecycle, server-side
// verification, header assembly with per-PJP quirk hooks, and response-code
// parsing — see docs/superpowers/specs/2026-09-10-go-snap-bi-core-design.md.
//
// Per-service bindings (in progress): typed request/response types built on
// Phase 1, added incrementally, one service at a time, each its own
// numbered phase. Implemented so far:
//   - BalanceInquiry (docs/superpowers/specs/2026-09-11-phase2-balance-inquiry-design.md)
//   - TransactionHistoryList (docs/superpowers/specs/2026-09-11-phase3-transaction-history-design.md)
//   - AccountCreation (docs/superpowers/specs/2026-09-11-phase4-account-creation-design.md)
//   - AccountBinding (docs/superpowers/specs/2026-09-11-phase5-account-binding-design.md)
//
// Most services (transfer, virtual account, QRIS, direct debit, etc.) are
// not yet implemented.
package snap
