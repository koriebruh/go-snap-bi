# Phase 16: Transaction Status Inquiry Bank

Source: `docs/research/2026-09-11-transfer-kredit-portal-research.md` §5.4
(lines 157-161), Service Code 36 — a standalone, single-endpoint
sub-group. This phase covers only this one endpoint, per advisor's
explicit recommendation not to batch it with Customer Top Up (§5.5):
different sub-group, different field table, and both sections mention
`partnerReferenceNo`-shaped fields, which risks the kind of
cross-endpoint type-split confusion that cost two extra review rounds
in Phase 14.

## Known package-wide gap discovered this phase (recorded, not fixed here)

Research §2 line 67: "ORIGIN / X-ORIGIN: hostname (Virtual Account +
Transaction Status Inquiry Bank sub-groups use "X-ORIGIN"; all other 8
sub-groups use bare "ORIGIN" — same portal, inconsistent header
name)." Verified against the current source: `header.go` line ~95
unconditionally does `h.Set("ORIGIN", b.Origin)` with no way to select
`X-ORIGIN` per call. This means:

- Every already-merged Virtual Account endpoint (Phases 13-15, 12
  functions) sends `ORIGIN` where the research doc says it should send
  `X-ORIGIN`.
- This phase's own `TransactionStatusInquiryBank` function has the
  same exposure — it cannot send `X-ORIGIN` either, since
  `HeaderBuilder` has no such option.

This is a real, verified, package-wide gap — not a hypothesis — but
fixing it requires changing `HeaderBuilder`'s shared header-emission
logic (adding some way to select the header name per call) and
touching all 12 already-merged VA functions plus this phase's new one.
That is a cross-cutting change out of scope for a single-endpoint
phase; it is recorded here as a known limitation for a future, scoped
bugfix phase, not attempted in this diff.

## Endpoint

- **36 Transaction Status Inquiry Bank** — POST, path
  `.../{version}/transaction-status-inquiry-bank` —
  `TransactionStatusInquiryBank(ctx, t, hb, req)`.

## Response shape: flat, not nested (read first)

Every prior Virtual Account response nested its data under
`virtualAccountData`/`virtualAccountdata`. This sub-group does not: per
§5.4's field list (no nested-object name is mentioned, unlike §5.3),
and per the precedent already in this package for a similarly-shaped
single-endpoint sub-group — `AccountInquiryInternalResponse`
(`account_inquiry_internal.go`, Phase 6) — fields sit directly on the
`Response` struct, no data sub-object. This phase follows that
precedent, not the VA nesting convention.

## Types

`TransactionStatusInquiryBankRequest`: per §5.4 line 159 —
`OriginalPartnerReferenceNo string` O, `OriginalReferenceNo string` O,
`OriginalExternalID string` O, `ServiceCode string` M (no omitempty;
String(2), e.g. "17" for Intrabank per the research doc's own example
— no worked example shows a non-string representation, so it stays
plain `string` per the ambiguous-type rule), `TransactionDate string`
O, `Amount *Money` O (Optional nested object → pointer, per the
established rule), `AdditionalInfo json.RawMessage` O.

`TransactionStatusInquiryBankResponse`: `ResponseCode string`,
`ResponseMessage string`, then every request field echoed back as
Optional (matching `AccountInquiryInternalResponse`'s
`PartnerReferenceNo` echo precedent) — `OriginalPartnerReferenceNo`,
`OriginalReferenceNo`, `OriginalExternalID`, `ServiceCode`,
`TransactionDate`, `Amount *Money` — plus §5.4 line 160's additions:
`BeneficiaryAccountNo string` M (no omitempty), `BeneficiaryBankCode
string` O, `PreviousResponseCode string` O, `ReferenceNumber string` M
(no omitempty), `SourceAccountNo string` M (no omitempty),
`TransactionID string` O, `LatestTransactionStatus string` M (no
omitempty — this is the shared `transactionStatus`/
`latestTransactionStatus` 2-digit enum documented in §3 line 79:
`00 Success, 01 Initiated, 02 Paying, 03 Pending, 04 Refunded,
05 Canceled, 06 Failed, 07 Not found`), `TransactionStatusDesc string`
O, `AdditionalInfo json.RawMessage` O.

## Function behavior

`TransactionStatusInquiryBank` follows the exact pattern already
established for every flat-response endpoint in the package (e.g.
`AccountInquiryInternal`): marshal `req`, set `hb.Body`, call `t.Do`,
`checkResponseStatus`, unmarshal into `Response`, error if
`ResponseCode == ""`. No method override — POST, matching every
endpoint in this section of the research doc.
