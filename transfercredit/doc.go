// Package transfercredit implements Bank Indonesia's SNAP "Transfer
// Kredit" API Service category (Service Codes 15-53, 75-78, 43
// endpoints): Account Inquiry, Trigger Transfer, Virtual Account,
// Transaction Status Inquiry (Bank), Customer Top Up, Bulk Cash In,
// Transfer to Bank, Transfer to OTC, and MPM/QR. See the parent
// github.com/koriebruh/go-snap-bi package's doc comment for the shared
// core types (HeaderBuilder, Transport, Money, the Err* sentinels)
// every calling function here depends on, and
// docs/research/2026-09-11-transfer-kredit-portal-research.md for the
// full endpoint field-table inventory this package is implemented
// against.
//
// # Known limitation: bare-number responseCode on two VA endpoints
//
// The Guides tab for VA - Inquiry VA (Service Code 30) and VA - Get
// Report (Service Code 35) documents responseCode rendered as a bare
// JSON number in some worked examples, rather than the quoted string
// every other endpoint in the package uses. This package always
// decodes responseCode into a Go string field; a server that actually
// sends the bare-number form on these two endpoints will fail to
// decode (returned as an error, never as silently-wrong data — see
// TestInquiryVA_BareNumberResponseCodeIsAKnownLimitation and
// TestVAGetReport_BareNumberResponseCodeIsAKnownLimitation for the
// pinned behavior). If you hit this in practice against a real PJP
// AIS's sandbox or production endpoint, please open an issue.
package transfercredit
