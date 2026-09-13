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
// Per-service request/response types and their calling functions live in
// two subpackages, split along SNAP's own top-level taxonomy:
//   - github.com/koriebruh/go-snap-bi/transferkredit — Transfer Kredit
//     (BalanceInquiry, Virtual Account, MPM/QR, Trigger Transfer, Bulk
//     Cash In, Transfer to Bank/OTC, Customer Top Up, and related
//     account/card/OTP endpoints)
//   - github.com/koriebruh/go-snap-bi/transferdebit — Transfer Debit
//     (Direct Debit, CPM, Direct Debit BI-FAST, Auth Payment)
//
// Both subpackages import this package (as snap) for the shared types
// above; internal/snaptest holds a test-only HeaderBuilder helper
// shared between them, not part of the public API.
//
// Per-service bindings were added incrementally, one service at a time,
// each its own numbered phase, before this package was split into the
// two subpackages above. Implemented so far (types below now live in
// transferkredit or transferdebit, not in this package):
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
//
// All of Transfer Kredit and all of Transfer Debit are now implemented
// (4 sub-groups — Direct Debit, CPM, Direct Debit BI-FAST, and Auth
// Payment — 21 endpoints total, per
// docs/research/2026-09-12-transfer-debit-portal-research.md). The
// remainder of SNAP's ~14 service groups (including Registrasi,
// Informasi Saldo, Riwayat Transaksi) beyond Transfer Kredit and
// Transfer Debit remains unresearched. See
// docs/research/2026-09-11-transfer-kredit-portal-research.md for the
// full Transfer Kredit endpoint inventory.
package snap
