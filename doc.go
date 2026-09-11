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
//   - AccountBindingInquiry, AccountUnbinding (docs/superpowers/specs/2026-09-11-phase6-account-binding-inquiry-unbinding-design.md)
//   - CardRegistration, CardRegistrationSetLimit (docs/superpowers/specs/2026-09-11-phase7-card-registration-set-limit-design.md)
//   - CardRegistrationInquiry (docs/superpowers/specs/2026-09-11-phase8-card-registration-inquiry-design.md)
//   - VerifyOTP, CardRegistrationUnbinding, OTP (docs/superpowers/specs/2026-09-11-phase9-verify-otp-unbinding-otp-design.md)
//   - AccountInquiryInternal, AccountInquiryExternal (docs/superpowers/specs/2026-09-11-phase10-account-inquiry-design.md)
//   - IntrabankTransfer, InterbankTransfer (docs/superpowers/specs/2026-09-11-phase11-trigger-transfer-intrabank-interbank-design.md)
//   - RequestForPayment, InterbankBulkTransfer, RTGSTransfer, SKNBITransfer,
//     plus inbound-only notification types for the latter three
//     (docs/superpowers/specs/2026-09-11-phase12-trigger-transfer-rfp-bulk-rtgs-sknbi-design.md)
//   - CreateVA, UpdateVA, UpdateStatusVA, InquiryVA, DeleteVA
//     (docs/superpowers/specs/2026-09-11-phase13-virtual-account-management-design.md)
//   - VAInquiry, VAPayment, VAInquiryStatus
//     (docs/superpowers/specs/2026-09-11-phase14-virtual-account-transaction-design.md)
//
// Most services (remaining Transfer Kredit sub-groups, Transfer Debit,
// etc.) are not yet implemented. See
// docs/research/2026-09-11-transfer-kredit-portal-research.md for the
// full Transfer Kredit endpoint inventory.
package snap
