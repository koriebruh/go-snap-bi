// Package snap implements Bank Indonesia's SNAP (Standar Nasional Open API
// Pembayaran) standard, document version 1.0.2 (September 2024).
//
// This package holds only the core primitives shared across every SNAP
// service: signing (SignSymmetric, SignAsymmetric, BuildStringToSignTransaction),
// token lifecycle (Token, TokenManager), server-side inbound-request
// verification (ServerVerifier, KeyStore), header assembly with per-PJP
// quirk hooks (HeaderBuilder), response-code parsing and the
// authoritative status check (ParseResponseCode, CheckResponseStatus,
// the Err* sentinels), the request/response transport (Transport,
// Envelope), and the shared Money amount type — see
// docs/superpowers/specs/2026-09-10-go-snap-bi-core-design.md (a
// development note kept locally only, not part of this module's
// published tree — every docs/ reference in this file is the same).
//
// # Package layout
//
// Per-service request/response types and their calling functions are
// split into subpackages that mirror the ASPI SNAP Developer Site's own
// top-level "API Services" categories 1:1 (apidevportal.aspi-indonesia.or.id/api-services)
// — so when the portal adds or changes an endpoint, which package to
// touch is never a judgment call:
//
//   - github.com/koriebruh/go-snap-bi/registration — Registrasi (Service
//     Codes 01-10, 81): CardRegistration and its Set Limit/Inquiry/
//     Unbinding variants, VerifyOTP, OTP, AccountCreation, AccountBinding
//     and its Inquiry/Unbinding variants, GetOAuthURL.
//   - github.com/koriebruh/go-snap-bi/balanceinfo — Informasi Saldo
//     (Service Code 11): BalanceInquiry. The portal's smallest category —
//     one endpoint, one file, one package, deliberately not folded into
//     a bigger one so it stays easy to find.
//   - github.com/koriebruh/go-snap-bi/transactionhistory — Riwayat
//     Transaksi (Service Codes 12-14): TransactionHistoryList,
//     TransactionHistoryDetail, BankStatement.
//   - github.com/koriebruh/go-snap-bi/transfercredit — Transfer Kredit
//     (Service Codes 15-53, 75-78, 43 endpoints — see
//     docs/research/2026-09-11-transfer-kredit-portal-research.md for
//     the full inventory): Account Inquiry, Trigger Transfer, Virtual
//     Account, Transaction Status Inquiry (Bank), Customer Top Up, Bulk
//     Cash In, Transfer to Bank, Transfer to OTC, MPM/QR, Transaction
//     Status Inquiry (non-bank).
//   - github.com/koriebruh/go-snap-bi/transferdebit — Transfer Debit
//     (Service Codes 54-72, 79-80, 21 endpoints — see
//     docs/research/2026-09-12-transfer-debit-portal-research.md):
//     Direct Debit, CPM, Direct Debit BI-FAST, Auth Payment.
//
// The portal's remaining two categories don't get their own subpackage:
//
//   - Keamanan (Service Codes 73-74, plus two sandbox-only "Signature
//     Auth"/"Signature Service" testing utilities that are not real
//     transactional endpoints and are out of scope): the Access Token
//     B2B/B2B2C endpoints are infrastructure every other package
//     depends on to get a token in the first place, not a peer domain
//     endpoint — they live in this package's own token.go
//     (TokenManager.AccessTokenB2B, TokenManager.AccessTokenB2B2C),
//     never in a subpackage.
//   - Administrasi has no API endpoints at all on the portal — it's
//     onboarding/account-management documentation only (IP allowlisting,
//     key management, registration paperwork). Nothing to implement,
//     and nothing will ever need to move here.
//
// Every subpackage imports this package (aliased "snap" regardless of
// its own package name, since the import path's basename doesn't match)
// for the shared types above; internal/snaptest holds a single test-only
// HeaderBuilder helper shared across all of them, not part of the
// public API.
//
// # Adding a new endpoint
//
// Find its Service Code, look up which of the 7 portal categories above
// it falls under, and add the file to that category's package (or
// this package's core, only for something Keamanan-shaped: shared
// authentication/signing infrastructure, not a per-service binding).
// Never split one portal category across two packages and never fold
// two categories into one, even if it looks like a small saving right
// now — that 1:1 mapping is the entire point of this layout.
//
// # Status
//
// All 7 of the portal's API Service categories are accounted for:
// Registrasi, Informasi Saldo, Riwayat Transaksi, Transfer Kredit, and
// Transfer Debit are fully implemented (79 endpoints total across the
// five); Keamanan's transactional endpoints live in this package's own
// token.go, as described above; Administrasi has no endpoints to
// implement. See docs/research/2026-09-11-transfer-kredit-portal-research.md
// and docs/research/2026-09-12-transfer-debit-portal-research.md for
// the Transfer Kredit and Transfer Debit endpoint inventories
// specifically, and CHANGELOG.md for the phase-by-phase history of how
// this package was built — both kept locally only (gitignored), not
// part of this module's published tree.
package snap
