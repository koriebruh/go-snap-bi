# Phase 19: Transfer To Bank

Source: `docs/research/2026-09-11-transfer-kredit-portal-research.md` §5.7
(lines 176-179), Service Codes 42-43, both endpoints in this
sub-group.

## Correct endpoint paths (read first — corrects a santa-loop-flagged gap)

Phase 18's santa-loop round 2 flagged that several earlier phases'
doc-comment endpoint paths didn't match the research doc — the actual
cause is that research §1's endpoint inventory table (lines 12-53)
carries the real per-endpoint paths, and earlier phases invented
readable slugs instead of citing it. This phase uses the table's
actual paths: **42** → `emoney/bank-account-inquiry`, **43** →
`emoney/transfer-bank` (table rows 30-31). Retrofitting the ~15 prior
phases' doc comments against this table is a package-wide cleanup, not
part of this phase — recorded as a known follow-up, not fixed here,
since no runtime behavior depends on a doc comment (callers always
supply `hb.EndpointURL` themselves).

## Open contradiction not resolved here (research §6 item 7)

§5.7 line 178 documents endpoint 42's own request field table using
`CustomerNumber` (capital C) — unlike every other endpoint in the
entire 43-endpoint corpus, which uses lowercase-first `customerNumber`.
No worked JSON example exists for this endpoint to break the tie. Per
the established practice for unresolved casing contradictions
(`bulkid`/`bulkId`, Phase 18), this phase models the literal,
only-available documented casing — `json:"CustomerNumber"` (capital C)
on `TransferToBankAccountInquiryRequest` only — rather than guessing
it's a typo and normalizing to lowercase. Endpoint 43's own field
listing uses lowercase `customerNumber` normally, so
`TransferToBankPaymentRequest.CustomerNumber` keeps the standard tag.

## Types

### TransferToBankAccountInquiry (42)

`TransferToBankAccountInquiryRequest`: `PartnerReferenceNo string` O +
`CustomerNumber string` `json:"CustomerNumber"` M (no omitempty — see
the casing note above) + `Amount Money` M (Mandatory nested object →
plain struct, per the established rule) + `BeneficiaryAccountNumber
string` O.

`TransferToBankAccountInquiryResponse`: `ResponseCode string`,
`ResponseMessage string`, `AccountType string` O + `BeneficiaryAccountNumber
string` M (no omitempty) + `BeneficiaryAccountName string` M (no
omitempty) + `BeneficiaryBankCode string` O + `BeneficiaryBankShortName
string` O + `BeneficiaryBankName string` O + `Amount Money` M
(Mandatory nested object → plain struct, same rule applied on the
response side) + `SessionID string` O.

### TransferToBankPayment (43)

`TransferToBankPaymentRequest`: `PartnerReferenceNo string` M (no
omitempty) + `CustomerNumber string` M (no omitempty; standard
lowercase tag, per this endpoint's own field listing) + `AccountType
string` O + `BeneficiaryAccountNumber string` M (no omitempty) +
`BeneficiaryBankCode string` O + `Amount Money` M (Mandatory nested
object → plain struct) + `SessionID string` O + `FeeType string` O.

`TransferToBankPaymentResponse`: `ResponseCode string`,
`ResponseMessage string`, `ReferenceNo string` C (omitempty) +
`TransactionDate string` O + `ReferenceNumber string` M (no omitempty —
research explicitly notes this is "a distinct field from `referenceNo`,
both present," matching the identical `ReferenceNo`/`ReferenceNumber`
coexistence pattern already established in Phase 17's
`CustomerTopUpResponse`).

## Function behavior

Both functions follow the exact pattern already established across
the package: marshal `req`, set `hb.Body`, call `t.Do`,
`checkResponseStatus`, unmarshal into `Response`, error if
`ResponseCode == ""`. No method override — POST, matching both
endpoints in this sub-group. `TransferToBankPayment` gets the standard
non-idempotency doc note (a mutating, state-changing call).
