# Phase 18: Bulk Cashin

Source: `docs/research/2026-09-11-transfer-kredit-portal-research.md` §5.6
(lines 170-175), Service Codes 40-41, both endpoints in this
sub-group.

## Slash notation (read first)

Research doc's `X/Y <letter>` notation means "X and Y are both
`<letter>`," not "either X or Y" — confirmed by precedent elsewhere in
the doc (e.g. §5.3 line 149: `partnerServiceId/customerNo/
virtualAccountNo/trxId M` means all four fields are Mandatory, not a
choice among them). Applied here: `bulkId/partnerBulkId M` (endpoint
41 request) means both `BulkID` and `PartnerBulkID` are Mandatory;
`referenceNo/partnerReferenceNo M` and `responseCode/responseMessage
M` (endpoint 41's per-item fields) mean all four are Mandatory on each
bulk item.

## Open contradiction not resolved here (research §6 item 6)

§5.6 line 172 documents endpoint 40's own field table with a
lowercase-d response field `bulkid` (not `bulkId`), flagged by the
research doc itself as "likely a typo... unresolved" — a lower
confidence level than the Virtual Account family's `virtualAccountdata`
casing, which research confirmed "not a table typo." No worked JSON
example exists for this endpoint to break the tie either way. Per the
established practice for unresolved research contradictions (VA
Inquiry Status's array-vs-object reading, VA Get Report's GET/POST),
this phase models the literal, only-available evidence — the field
table's `bulkid` — rather than guessing at the "probably a typo"
correction, and documents the uncertainty in the type's doc comment.
Endpoint 41's own field description independently uses camelCase
`bulkId` for its own response, so the two endpoints in this same
sub-group get different literal tags: `bulkid` (40) and `bulkId` (41).

## Types

### SubmitBulkCashIn (40)

`BulkCashInItem` (one entry in `bulkObject[]`): `AccountNumber string`
M (no omitempty) + `AccountName string` O + `Amount *Money` O +
`PartnerReferenceNo string` M (no omitempty) + `AdditionalInfo
json.RawMessage` O.

`SubmitBulkCashInRequest`: `PartnerBulkID string` O + `TransactionDate
string` M (no omitempty) + `Currency string` O + `BulkObject
[]BulkCashInItem` O (the array's own cardinality is unmarked in §5.6;
treated as Optional per the package's default for unmarked container
fields, matching how Phase 11-13 handled unmarked response fields) +
`FeeType string` O + `AdditionalInfo json.RawMessage` O.

`SubmitBulkCashInResponse`: `ResponseCode string`, `ResponseMessage
string`, `BulkID string` `json:"bulkid"` M (no omitempty — see the
casing note above) + `PartnerBulkID string` O.

### NotifyBulkCashIn (41)

`BulkCashInNotificationItem` (one entry in `bulkObject[]`) —
deliberately its own type, not a reuse of Phase 12's
`InterbankBulkTransferNotificationItem` (which has only 3 fields vs.
this type's 7), per the package's "distinct types per service code"
convention: `CustomerNumber string` M (no omitempty) + `CustomerName
string` O + `Amount *Money` O + `ReferenceNo string` M (no omitempty)
+ `PartnerReferenceNo string` M (no omitempty) + `ResponseCode string`
M (no omitempty; per-item settlement result, distinct from the
envelope's own `ResponseCode`) + `ResponseMessage string` M (no
omitempty) + `AdditionalInfo json.RawMessage` O.

`NotifyBulkCashInRequest`: `BulkID string` M (no omitempty) +
`PartnerBulkID string` M (no omitempty) + `BulkObject
[]BulkCashInNotificationItem` O (array cardinality unmarked, same
default as above).

`NotifyBulkCashInResponse`: `ResponseCode string`, `ResponseMessage
string`, `BulkID string` `json:"bulkId"` M (no omitempty — camelCase,
per this endpoint's own field description) + `PartnerBulkID string` M
(no omitempty).

## Function behavior

Both functions follow the exact pattern already established across
the package: marshal `req`, set `hb.Body`, call `t.Do`,
`checkResponseStatus`, unmarshal into `Response`, error if
`ResponseCode == ""`. No method override — POST, matching every
endpoint in this sub-group. `SubmitBulkCashIn` and `NotifyBulkCashIn`
both get the standard non-idempotency doc note (mutating,
state-changing calls), matching every other mutating endpoint in the
package.
