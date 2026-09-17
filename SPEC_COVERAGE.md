# Spec Coverage

Per-endpoint traceability from the SNAP Technical Specification v1.0.2
(September 2024) to this SDK's implementation: every Service Code the
ASPI SNAP Developer Site publishes, mapped to the Go package and
function/type that implements it, and whether that binding has its own
test file exercising it. Generated from the source tree on 2026-09-17.

This is a self-reported traceability artifact, not an independent or
official SNAP conformance verification — it documents what this
project claims to implement and lets a reviewer check that claim
against the actual source, rather than taking an endpoint count on
faith. See [CONFORMANCE.md](./CONFORMANCE.md) for the test-suite
summary this "Tests" column is drawn from.

"Tests" marks ✓ when a `_test.go` file exists alongside the
implementation and references the listed function or type by name — it
does not re-verify the tests still pass; see CONFORMANCE.md and CI for
that. Rows marked *(type only)* are inbound webhook payloads (SNAP
pushes these to the integrator, this SDK never calls them) — they have
a request type to unmarshal into and no response type.


### Registrasi (`registration`)

| Service Code | SNAP Operation | Go Function | Request Type | Response Type | Tests |
|---|---|---|---|---|---|
| 01 | Card Registration | `CardRegistration` | `CardRegistrationRequest` | `CardRegistrationResponse` | ✓ |
| 02 | Card Registration Set Limit | `CardRegistrationSetLimit` | `CardRegistrationSetLimitRequest` | `CardRegistrationSetLimitResponse` | ✓ |
| 03 | Card Registration Inquiry | `CardRegistrationInquiry` | `string` | `CardRegistrationInquiryResponse` | ✓ |
| 04 | Verify OTP | `VerifyOTP` | `VerifyOTPRequest` | `VerifyOTPResponse` | ✓ |
| 05 | Card Registration Unbinding | `CardRegistrationUnbinding` | `CardRegistrationUnbindingRequest` | `CardRegistrationUnbindingResponse` | ✓ |
| 06 | Account Creation | `AccountCreation` | `AccountCreationRequest` | `AccountCreationResponse` | ✓ |
| 07 | Account Binding | `AccountBinding` | `AccountBindingRequest` | `AccountBindingResponse` | ✓ |
| 08 | Account Binding Inquiry | `AccountBindingInquiry` | `AccountBindingInquiryRequest` | `AccountBindingInquiryResponse` | ✓ |
| 09 | Account Unbinding | `AccountUnbinding` | `AccountUnbindingRequest` | `AccountUnbindingResponse` | ✓ |
| 10 | Get OAuth URL | `GetOAuthURL` | `GetOAuthURLRequest` | `GetOAuthURLResponse` | ✓ |
| 81 | OTP | `OTP` | `OTPRequest` | `OTPResponse` | ✓ |

### Informasi Saldo (`balanceinfo`)

| Service Code | SNAP Operation | Go Function | Request Type | Response Type | Tests |
|---|---|---|---|---|---|
| 11 | Balance Inquiry | `BalanceInquiry` | `BalanceInquiryRequest` | `BalanceInquiryResponse` | ✓ |

### Riwayat Transaksi (`transactionhistory`)

| Service Code | SNAP Operation | Go Function | Request Type | Response Type | Tests |
|---|---|---|---|---|---|
| 12 | Transaction History List | `TransactionHistoryList` | `TransactionHistoryListRequest` | `TransactionHistoryListResponse` | ✓ |
| 13 | Transaction History Detail | `TransactionHistoryDetail` | `TransactionHistoryDetailRequest` | `TransactionHistoryDetailResponse` | ✓ |
| 14 | Bank Statement | `BankStatement` | `BankStatementRequest` | `BankStatementResponse` | ✓ |

### Transfer Kredit (`transfercredit`)

| Service Code | SNAP Operation | Go Function | Request Type | Response Type | Tests |
|---|---|---|---|---|---|
| 15 | Account Inquiry Internal | `AccountInquiryInternal` | `AccountInquiryInternalRequest` | `AccountInquiryInternalResponse` | ✓ |
| 16 | Account Inquiry External | `AccountInquiryExternal` | `AccountInquiryExternalRequest` | `AccountInquiryExternalResponse` | ✓ |
| 17 | Intrabank Transfer | `IntrabankTransfer` | `IntrabankTransferRequest` | `IntrabankTransferResponse` | ✓ |
| 18 | Interbank Transfer | `InterbankTransfer` | `InterbankTransferRequest` | `InterbankTransferResponse` | ✓ |
| 19 | Request For Payment | `RequestForPayment` | `RequestForPaymentRequest` | `RequestForPaymentResponse` | ✓ |
| 20 | Interbank Bulk Transfer | `InterbankBulkTransfer` | `InterbankBulkTransferRequest` | `InterbankBulkTransferResponse` | ✓ |
| 21 | Interbank Bulk Transfer Notification | *(type only)* | `InterbankBulkTransferNotificationRequest` | `—` | ✓ |
| 22 | RTGS Transfer | `RTGSTransfer` | `RTGSTransferRequest` | `RTGSTransferResponse` | ✓ |
| 23 | SKNBI Transfer | `SKNBITransfer` | `SKNBITransferRequest` | `SKNBITransferResponse` | ✓ |
| 24 | VA Inquiry | `VAInquiry` | `VAInquiryRequest` | `VAInquiryResponse` | ✓ |
| 25 | VA Payment | `VAPayment` | `VAPaymentRequest` | `VAPaymentResponse` | ✓ |
| 26 | VA Inquiry Status | `VAInquiryStatus` | `VAInquiryStatusRequest` | `VAInquiryStatusResponse` | ✓ |
| 27 | Create VA | `CreateVA` | `CreateVARequest` | `CreateVAResponse` | ✓ |
| 28 | Update VA | `UpdateVA` | `UpdateVARequest` | `UpdateVAResponse` | ✓ |
| 29 | Update Status VA | `UpdateStatusVA` | `UpdateStatusVARequest` | `UpdateStatusVAResponse` | ✓ |
| 30 | Inquiry VA | `InquiryVA` | `InquiryVARequest` | `InquiryVAResponse` | ✓ |
| 31 | Delete VA | `DeleteVA` | `DeleteVARequest` | `DeleteVAResponse` | ✓ |
| 32 | VA Inquiry Payment Intrabank | `VAInquiryPaymentIntrabank` | `VAInquiryPaymentIntrabankRequest` | `VAInquiryPaymentIntrabankResponse` | ✓ |
| 33 | VA Payment Intrabank | `VAPaymentIntrabank` | `VAPaymentIntrabankRequest` | `VAPaymentIntrabankResponse` | ✓ |
| 34 | VA Notify Payment Intrabank | `VANotifyPaymentIntrabank` | `VANotifyPaymentIntrabankRequest` | `VANotifyPaymentIntrabankResponse` | ✓ |
| 35 | VA Get Report | `VAGetReport` | `VAGetReportRequest` | `VAGetReportResponse` | ✓ |
| 36 | Transaction Status Inquiry Bank | `TransactionStatusInquiryBank` | `TransactionStatusInquiryBankRequest` | `TransactionStatusInquiryBankResponse` | ✓ |
| 37 | Account Inquiry Customer Top Up | `AccountInquiryCustomerTopUp` | `AccountInquiryCustomerTopUpRequest` | `AccountInquiryCustomerTopUpResponse` | ✓ |
| 38 | Customer Top Up | `CustomerTopUp` | `CustomerTopUpRequest` | `CustomerTopUpResponse` | ✓ |
| 39 | Customer Top Up Inquiry Status | `CustomerTopUpInquiryStatus` | `CustomerTopUpInquiryStatusRequest` | `CustomerTopUpInquiryStatusResponse` | ✓ |
| 40 | Submit Bulk Cash In | `SubmitBulkCashIn` | `SubmitBulkCashInRequest` | `SubmitBulkCashInResponse` | ✓ |
| 41 | Notify Bulk Cash In | *(type only)* | `NotifyBulkCashInRequest` | `—` | ✓ |
| 42 | Transfer To Bank Account Inquiry | `TransferToBankAccountInquiry` | `TransferToBankAccountInquiryRequest` | `TransferToBankAccountInquiryResponse` | ✓ |
| 43 | Transfer To Bank Payment | `TransferToBankPayment` | `TransferToBankPaymentRequest` | `TransferToBankPaymentResponse` | ✓ |
| 44 | Transfer To OTC Create Payment | `TransferToOTCCreatePayment` | `TransferToOTCCreatePaymentRequest` | `TransferToOTCCreatePaymentResponse` | ✓ |
| 45 | Transfer To OTC Transfer Status | `TransferToOTCTransferStatus` | `TransferToOTCTransferStatusRequest` | `TransferToOTCTransferStatusResponse` | ✓ |
| 46 | Transfer To OTC Cancel Payment | `TransferToOTCCancelPayment` | `TransferToOTCCancelPaymentRequest` | `TransferToOTCCancelPaymentResponse` | ✓ |
| 47 | Generate QRMPM | `GenerateQRMPM` | `GenerateQRMPMRequest` | `GenerateQRMPMResponse` | ✓ |
| 48 | Decode QRMPM | `DecodeQRMPM` | `DecodeQRMPMRequest` | `DecodeQRMPMResponse` | ✓ |
| 49 | Apply OTT | `ApplyOTT` | `ApplyOTTRequest` | `ApplyOTTResponse` | ✓ |
| 50 | QRMPM Payment H2H | `QRMPMPaymentH2H` | `QRMPMPaymentH2HRequest` | `QRMPMPaymentH2HResponse` | ✓ |
| 51 | QRMPM Query Payment | `QRMPMQueryPayment` | `QRMPMQueryPaymentRequest` | `QRMPMQueryPaymentResponse` | ✓ |
| 52 | QRMPM Payment Notification | *(type only)* | `QRMPMPaymentNotificationRequest` | `—` | ✓ |
| 53 | Transaction Status Inquiry Non Bank | `TransactionStatusInquiryNonBank` | `TransactionStatusInquiryNonBankRequest` | `TransactionStatusInquiryNonBankResponse` | ✓ |
| 75 | SKNBI Notification | *(type only)* | `SKNBINotificationRequest` | `—` | ✓ |
| 76 | RTGS Notification | *(type only)* | `RTGSNotificationRequest` | `—` | ✓ |
| 77 | QRMPM Cancel Payment | `QRMPMCancelPayment` | `QRMPMCancelPaymentRequest` | `QRMPMCancelPaymentResponse` | ✓ |
| 78 | QRMPM Refund Payment | `QRMPMRefundPayment` | `QRMPMRefundPaymentRequest` | `QRMPMRefundPaymentResponse` | ✓ |

### Transfer Debit (`transferdebit`)

| Service Code | SNAP Operation | Go Function | Request Type | Response Type | Tests |
|---|---|---|---|---|---|
| 54 | Direct Debit Payment | `DirectDebitPayment` | `DirectDebitPaymentRequest` | `DirectDebitPaymentResponse` | ✓ |
| 55 | Direct Debit Payment Status | `DirectDebitPaymentStatus` | `DirectDebitPaymentStatusRequest` | `DirectDebitPaymentStatusResponse` | ✓ |
| 56 | Direct Debit Payment Notification | *(type only)* | `DirectDebitPaymentNotificationRequest` | `—` | ✓ |
| 57 | Direct Debit Payment Cancel | `DirectDebitPaymentCancel` | `DirectDebitPaymentCancelRequest` | `DirectDebitPaymentCancelResponse` | ✓ |
| 58 | Direct Debit Payment Refund | `DirectDebitPaymentRefund` | `DirectDebitPaymentRefundRequest` | `DirectDebitPaymentRefundResponse` | ✓ |
| 59 | CPM Generate QR | `CPMGenerateQR` | `CPMGenerateQRRequest` | `CPMGenerateQRResponse` | ✓ |
| 60 | CPM Payment | `CPMPayment` | `CPMPaymentRequest` | `CPMPaymentResponse` | ✓ |
| 61 | CPM Query Payment | `CPMQueryPayment` | `CPMQueryPaymentRequest` | `CPMQueryPaymentResponse` | ✓ |
| 62 | CPM Cancel Payment | `CPMCancelPayment` | `CPMCancelPaymentRequest` | `CPMCancelPaymentResponse` | ✓ |
| 63 | Auth Payment | `AuthPayment` | `AuthPaymentRequest` | `AuthPaymentResponse` | ✓ |
| 64 | Auth Payment Query | `AuthPaymentQuery` | `AuthPaymentQueryRequest` | `AuthPaymentQueryResponse` | ✓ |
| 65 | Auth Capture | `AuthCapture` | `AuthCaptureRequest` | `AuthCaptureResponse` | ✓ |
| 66 | Auth Capture Query | `AuthCaptureQuery` | `AuthCaptureQueryRequest` | `AuthCaptureQueryResponse` | ✓ |
| 67 | Auth Void | `AuthVoid` | `AuthVoidRequest` | `AuthVoidResponse` | ✓ |
| 68 | Auth Void Query | `AuthVoidQuery` | `AuthVoidQueryRequest` | `AuthVoidQueryResponse` | ✓ |
| 69 | Auth Refund | `AuthRefund` | `AuthRefundRequest` | `AuthRefundResponse` | ✓ |
| 70 | Direct Debit BI-FAST E-Mandate Registration | `DirectDebitBIFASTEMandateRegistration` | `DirectDebitBIFASTEMandateRegistrationRequest` | `DirectDebitBIFASTEMandateRegistrationResponse` | ✓ |
| 71 | Direct Debit BI-FAST Payment | `DirectDebitBIFASTPayment` | `DirectDebitBIFASTPaymentRequest` | `DirectDebitBIFASTPaymentResponse` | ✓ |
| 72 | Direct Debit BI-FAST Notification | *(type only)* | `DirectDebitBIFASTNotificationRequest` | `—` | ✓ |
| 79 | CPM Payment Notification | *(type only)* | `CPMPaymentNotificationRequest` | `—` | ✓ |
| 80 | CPM Refund Payment | `CPMRefundPayment` | `CPMRefundPaymentRequest` | `CPMRefundPaymentResponse` | ✓ |


### Keamanan (root `snap` package, `token.go`)

| Service Code | SNAP Operation | Go Method | Tests |
|---|---|---|---|
| 73 | Access Token (B2B) | `TokenManager.AccessTokenB2B` | ✓ |
| 74 | Access Token (B2B2C) | `TokenManager.AccessTokenB2B2C` | ✓ |

Keamanan's two sandbox-only "Signature Auth"/"Signature Service"
testing utilities are not real transactional endpoints and are
explicitly out of scope (see `doc.go`).

### Administrasi

No API endpoints on the portal — onboarding/account-management
documentation only (IP allowlisting, key management, registration
paperwork). Nothing to implement, nothing to trace here.

## Total

79 endpoint bindings across 5 domain packages (71 outbound calling
functions + 8 inbound-only notification types), plus 2 Access Token
methods in the root package — 81 Service Codes accounted for.
