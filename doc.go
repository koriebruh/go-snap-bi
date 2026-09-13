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
// docs/superpowers/specs/2026-09-10-go-snap-bi-core-design.md.
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
// # History
//
// Per-service bindings were added incrementally, one service at a time,
// each its own numbered phase, before this package was split into
// subpackages. The list below is that history — every listed type now
// lives in one of the subpackages above, not in this package, per the
// mapping described above:
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
//   - VAInquiryPaymentIntrabank, VAPaymentIntrabank,
//     VANotifyPaymentIntrabank, VAGetReport (completes the Virtual
//     Account sub-group)
//     (docs/superpowers/specs/2026-09-11-phase15-virtual-account-intrabank-report-design.md)
//   - TransactionStatusInquiryBank
//     (docs/superpowers/specs/2026-09-11-phase16-transaction-status-inquiry-bank-design.md)
//   - AccountInquiryCustomerTopUp, CustomerTopUp, CustomerTopUpInquiryStatus
//     (docs/superpowers/specs/2026-09-11-phase17-customer-top-up-design.md)
//   - SubmitBulkCashIn, plus inbound-only notification types for
//     Notify Bulk Cash In
//     (docs/superpowers/specs/2026-09-11-phase18-bulk-cashin-design.md)
//   - TransferToBankAccountInquiry, TransferToBankPayment
//     (docs/superpowers/specs/2026-09-11-phase19-transfer-to-bank-design.md)
//   - TransferToOTCCreatePayment, TransferToOTCTransferStatus,
//     TransferToOTCCancelPayment
//     (docs/superpowers/specs/2026-09-11-phase20-transfer-to-otc-design.md)
//   - GenerateQRMPM, DecodeQRMPM
//     (docs/superpowers/specs/2026-09-11-phase21-mpm-qr-generate-decode-design.md)
//   - ApplyOTT, QRMPMPaymentH2H
//     (docs/superpowers/specs/2026-09-11-phase22-mpm-qr-apply-ott-payment-h2h-design.md)
//   - QRMPMQueryPayment, plus inbound-only notification types for
//     Payment Notification
//     (docs/superpowers/specs/2026-09-11-phase23-mpm-qr-query-payment-notification-design.md)
//   - QRMPMCancelPayment, QRMPMRefundPayment (completes the MPM/QR
//     sub-group)
//     (docs/superpowers/specs/2026-09-11-phase24-mpm-qr-cancel-refund-design.md)
//   - TransactionStatusInquiryNonBank (completes Transfer Kredit)
//     (docs/superpowers/specs/2026-09-11-phase25-transaction-status-inquiry-nonbank-design.md)
//   - DirectDebitPayment, DirectDebitPaymentStatus
//     (docs/superpowers/specs/2026-09-12-phase26-direct-debit-payment-status-design.md)
//   - DirectDebitPaymentCancel, DirectDebitPaymentRefund, plus an
//     inbound-only notification type for Direct Debit Payment
//     Notification (completes the Direct Debit sub-group)
//     (docs/superpowers/specs/2026-09-12-phase27-direct-debit-notify-cancel-refund-design.md)
//   - CPMGenerateQR, CPMPayment
//     (docs/superpowers/specs/2026-09-13-phase28-cpm-generate-payment-design.md)
//   - CPMQueryPayment, CPMCancelPayment, CPMRefundPayment, plus an
//     inbound-only notification type for Payment Notification
//     (completes the CPM sub-group)
//     (docs/superpowers/specs/2026-09-13-phase29-cpm-query-cancel-notify-refund-design.md)
//   - DirectDebitBIFASTEMandateRegistration, DirectDebitBIFASTPayment,
//     plus an inbound-only notification type for Notify (completes
//     Direct Debit BI-FAST; only Auth Payment remains to complete
//     Transfer Debit)
//     (docs/superpowers/specs/2026-09-13-phase30-direct-debit-bifast-design.md)
//   - AuthPayment, AuthPaymentQuery (Auth Payment sub-group, part 1 of 3)
//     (docs/superpowers/specs/2026-09-13-phase31-auth-payment-query-design.md)
//   - AuthCapture, AuthCaptureQuery, AuthVoid, AuthVoidQuery (Auth
//     Payment sub-group, part 2 of 3)
//     (docs/superpowers/specs/2026-09-13-phase32-auth-capture-void-design.md)
//   - AuthRefund (completes the Auth Payment sub-group and, with it,
//     all of Transfer Debit)
//     (docs/superpowers/specs/2026-09-13-phase33-auth-refund-design.md)
//   - GetOAuthURL (Registrasi, completing that category's remaining
//     endpoint)
//   - TransactionHistoryDetail, BankStatement (Riwayat Transaksi,
//     completing that category)
//
// All 7 of the portal's API Services categories are now accounted for:
// Registrasi, Informasi Saldo, Riwayat Transaksi, Transfer Kredit, and
// Transfer Debit are fully implemented (79 endpoints total across the
// five); Keamanan's transactional endpoints live in this package's own
// token.go, as described above; Administrasi has no endpoints to
// implement. See docs/research/2026-09-11-transfer-kredit-portal-research.md
// and docs/research/2026-09-12-transfer-debit-portal-research.md for
// the Transfer Kredit and Transfer Debit endpoint inventories
// specifically.
package snap
