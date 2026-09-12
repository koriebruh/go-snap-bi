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
//
// All of Transfer Kredit is now implemented. Transfer Debit (4
// sub-groups including Direct Debit and QR/CPM, per
// docs/superpowers/specs/2026-09-10-go-snap-bi-core-design.md) remains
// entirely unresearched and needs its own research pass before any
// design doc can be written for it, as does the remainder of SNAP's
// non-Transfer-Kredit service groups (Registrasi, Informasi Saldo,
// Riwayat Transaksi) beyond the handful of endpoints already
// implemented above. See
// docs/research/2026-09-11-transfer-kredit-portal-research.md for the
// full Transfer Kredit endpoint inventory.
package snap
