// Package snap implements Bank Indonesia's SNAP (Standar Nasional Open API
// Pembayaran) standard, document version 1.0.2 (September 2024).
//
// Phase 1 (complete): signing primitives, token lifecycle, server-side
// verification, header assembly with per-PJP quirk hooks, and response-code
// parsing — see docs/superpowers/specs/2026-09-10-go-snap-bi-core-design.md.
//
// Per-service bindings (in progress): typed request/response types built on
// Phase 1, added incrementally, one service at a time, each its own
// numbered phase, each with a spec at docs/superpowers/specs/. Implemented
// so far (this list has been missed on phase landing 3 times already —
// check it explicitly, don't assume the last editor remembered):
//   - Phase 2: BalanceInquiry
//   - Phase 3: TransactionHistoryList
//   - Phase 4: AccountCreation
//   - Phase 5: AccountBinding
//
// Most services (transfer, virtual account, QRIS, direct debit, etc.) are
// not yet implemented.
package snap
