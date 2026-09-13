// Package transferdebit implements Bank Indonesia's SNAP "Transfer
// Debit" API Service category (Service Codes 54-72, 79-80, 21
// endpoints): Direct Debit, CPM, Direct Debit BI-FAST, and Auth
// Payment. See the parent github.com/koriebruh/go-snap-bi package's
// doc comment for the shared core types (HeaderBuilder, Transport,
// Money, the Err* sentinels) every calling function here depends on,
// and docs/research/2026-09-12-transfer-debit-portal-research.md for
// the full endpoint field-table inventory this package is implemented
// against.
package transferdebit
