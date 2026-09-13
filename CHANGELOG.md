# Changelog

Per-service bindings were added incrementally, one Service Code at a
time, each its own numbered phase, before this package was split into
the subpackages described in [`doc.go`](./doc.go). This is that
history — every type listed below now lives in one of the subpackages
(`registration`, `balanceinfo`, `transactionhistory`, `transfercredit`,
`transferdebit`), not in the root `snap` package.

- BalanceInquiry (docs/superpowers/specs/2026-09-11-phase2-balance-inquiry-design.md)
- TransactionHistoryList (docs/superpowers/specs/2026-09-11-phase3-transaction-history-design.md)
- AccountCreation (docs/superpowers/specs/2026-09-11-phase4-account-creation-design.md)
- AccountBinding (docs/superpowers/specs/2026-09-11-phase5-account-binding-design.md)
- AccountBindingInquiry, AccountUnbinding (docs/superpowers/specs/2026-09-11-phase6-account-binding-inquiry-unbinding-design.md)
- CardRegistration, CardRegistrationSetLimit (docs/superpowers/specs/2026-09-11-phase7-card-registration-set-limit-design.md)
- CardRegistrationInquiry (docs/superpowers/specs/2026-09-11-phase8-card-registration-inquiry-design.md)
- VerifyOTP, CardRegistrationUnbinding, OTP (docs/superpowers/specs/2026-09-11-phase9-verify-otp-unbinding-otp-design.md)
- AccountInquiryInternal, AccountInquiryExternal (docs/superpowers/specs/2026-09-11-phase10-account-inquiry-design.md)
- IntrabankTransfer, InterbankTransfer (docs/superpowers/specs/2026-09-11-phase11-trigger-transfer-intrabank-interbank-design.md)
- RequestForPayment, InterbankBulkTransfer, RTGSTransfer, SKNBITransfer,
  plus inbound-only notification types for the latter three
  (docs/superpowers/specs/2026-09-11-phase12-trigger-transfer-rfp-bulk-rtgs-sknbi-design.md)
- CreateVA, UpdateVA, UpdateStatusVA, InquiryVA, DeleteVA
  (docs/superpowers/specs/2026-09-11-phase13-virtual-account-management-design.md)
- VAInquiry, VAPayment, VAInquiryStatus
  (docs/superpowers/specs/2026-09-11-phase14-virtual-account-transaction-design.md)
- VAInquiryPaymentIntrabank, VAPaymentIntrabank, VANotifyPaymentIntrabank,
  VAGetReport (completes the Virtual Account sub-group)
  (docs/superpowers/specs/2026-09-11-phase15-virtual-account-intrabank-report-design.md)
- TransactionStatusInquiryBank
  (docs/superpowers/specs/2026-09-11-phase16-transaction-status-inquiry-bank-design.md)
- AccountInquiryCustomerTopUp, CustomerTopUp, CustomerTopUpInquiryStatus
  (docs/superpowers/specs/2026-09-11-phase17-customer-top-up-design.md)
- SubmitBulkCashIn, plus inbound-only notification types for Notify Bulk
  Cash In
  (docs/superpowers/specs/2026-09-11-phase18-bulk-cashin-design.md)
- TransferToBankAccountInquiry, TransferToBankPayment
  (docs/superpowers/specs/2026-09-11-phase19-transfer-to-bank-design.md)
- TransferToOTCCreatePayment, TransferToOTCTransferStatus,
  TransferToOTCCancelPayment
  (docs/superpowers/specs/2026-09-11-phase20-transfer-to-otc-design.md)
- GenerateQRMPM, DecodeQRMPM
  (docs/superpowers/specs/2026-09-11-phase21-mpm-qr-generate-decode-design.md)
- ApplyOTT, QRMPMPaymentH2H
  (docs/superpowers/specs/2026-09-11-phase22-mpm-qr-apply-ott-payment-h2h-design.md)
- QRMPMQueryPayment, plus inbound-only notification types for Payment
  Notification
  (docs/superpowers/specs/2026-09-11-phase23-mpm-qr-query-payment-notification-design.md)
- QRMPMCancelPayment, QRMPMRefundPayment (completes the MPM/QR
  sub-group)
  (docs/superpowers/specs/2026-09-11-phase24-mpm-qr-cancel-refund-design.md)
- TransactionStatusInquiryNonBank (completes Transfer Kredit)
  (docs/superpowers/specs/2026-09-11-phase25-transaction-status-inquiry-nonbank-design.md)
- DirectDebitPayment, DirectDebitPaymentStatus
  (docs/superpowers/specs/2026-09-12-phase26-direct-debit-payment-status-design.md)
- DirectDebitPaymentCancel, DirectDebitPaymentRefund, plus an
  inbound-only notification type for Direct Debit Payment Notification
  (completes the Direct Debit sub-group)
  (docs/superpowers/specs/2026-09-12-phase27-direct-debit-notify-cancel-refund-design.md)
- CPMGenerateQR, CPMPayment
  (docs/superpowers/specs/2026-09-13-phase28-cpm-generate-payment-design.md)
- CPMQueryPayment, CPMCancelPayment, CPMRefundPayment, plus an
  inbound-only notification type for Payment Notification (completes
  the CPM sub-group)
  (docs/superpowers/specs/2026-09-13-phase29-cpm-query-cancel-notify-refund-design.md)
- DirectDebitBIFASTEMandateRegistration, DirectDebitBIFASTPayment, plus
  an inbound-only notification type for Notify (completes Direct Debit
  BI-FAST)
  (docs/superpowers/specs/2026-09-13-phase30-direct-debit-bifast-design.md)
- AuthPayment, AuthPaymentQuery (Auth Payment sub-group, part 1 of 3)
  (docs/superpowers/specs/2026-09-13-phase31-auth-payment-query-design.md)
- AuthCapture, AuthCaptureQuery, AuthVoid, AuthVoidQuery (Auth Payment
  sub-group, part 2 of 3)
  (docs/superpowers/specs/2026-09-13-phase32-auth-capture-void-design.md)
- AuthRefund (completes the Auth Payment sub-group and, with it, all of
  Transfer Debit)
  (docs/superpowers/specs/2026-09-13-phase33-auth-refund-design.md)
- GetOAuthURL (Registrasi, completing that category's remaining
  endpoint)
- TransactionHistoryDetail, BankStatement (Riwayat Transaksi, completing
  that category)
- Package layout split into `registration`, `balanceinfo`,
  `transactionhistory`, `transfercredit` (renamed from `transferkredit`),
  and `transferdebit`, one subpackage per ASPI portal API Service
  category.

All 7 of the portal's API Service categories are now accounted for:
Registrasi, Informasi Saldo, Riwayat Transaksi, Transfer Kredit, and
Transfer Debit are fully implemented (79 endpoints total across the
five); Keamanan's transactional endpoints live in the root package's
`token.go`; Administrasi has no endpoints to implement.
