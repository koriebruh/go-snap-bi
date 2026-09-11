# Phase 17: Customer Top Up

Source: `docs/research/2026-09-11-transfer-kredit-portal-research.md` §5.5
(lines 162-167), Service Codes 37-39, all 3 endpoints in this
sub-group.

## Ambiguous-type decisions (this phase)

| Field | Endpoint(s) | Guides label | Worked example (§5.5) | Go type |
|---|---|---|---|---|
| `customerMonthlyInLimit` | 37 (response) | Numeric | "quoted string on wire" per research | `json.RawMessage` — same rule as `BillReferenceNo` (Phase 13): a documented Numeric field gets `json.RawMessage` even when the one worked example shows it quoted, since one example is thin evidence about every issuer |
| `categoryId` | 38 (request) | Numeric | "quoted string on wire" per research | `json.RawMessage`, same reasoning |
| `customerNumber` | 37, 38 | String(32) request / String(64) response | no non-string example noted (only a length mismatch between request and response, noted below) | `string` |

## Request/response length mismatch (recorded, not resolved)

§5.5 line 164 notes endpoint 37's `customerNumber` is String(32) on the
request but String(64) on the response (masked, e.g.
`"XXXXXXXXX1857"`). Both are modeled as plain `string` — Go strings
have no length constraint, so this is a documentation-only distinction
with no code consequence.

## Endpoint 38's undocumented response field

§5.5 line 165: endpoint 38's own worked response includes a
`referenceNumber` field ("REF993883") absent from the field table.
Per the package's practice of trusting worked examples over an
incomplete table (e.g. `BillDetail`'s fields), `ReferenceNumber string`
(Optional) is added to `CustomerTopUpResponse` even though it has no
table entry — this is an empirically observed field, not a guess.

## Endpoint 39: reuses Phase 16's shape under new names

§5.5 line 166: "Customer Top Up Inquiry Status (39): same
originalX/serviceCode/status shape as Transaction Status Inquiry
Bank." Per the "distinct types per service code even for identical
shapes" convention (Phase 12 RTGS/SKNBI, Phase 13 VA data types,
Phase 15 `GetReportData`), this phase defines its own
`CustomerTopUpInquiryStatusRequest`/`Response` types, field-identical
to Phase 16's `TransactionStatusInquiryBankRequest`/`Response`, guarded
by a full-equality reflection test (same pattern as Phase 15's
`GetReportData` ≡ `VAInquiryStatusData` guard).

## Types

### AccountInquiryCustomerTopUp (37)

Portal title "Account Inquiry - Customer Top Up"; named
`AccountInquiryCustomerTopUp` to avoid colliding with the existing
`AccountInquiryInternal`/`AccountInquiryExternal` (Phase 6, a different
sub-group).

`AccountInquiryCustomerTopUpRequest`: `PartnerReferenceNo string` O +
`CustomerNumber string` C (omitempty; "mandatory if B2B2C token null" —
a data-dependent condition the type system can't express, so Optional
here, same handling as Phase 15's Conditional fields) + `Amount Money`
M (Mandatory nested object → plain struct, per the established rule) +
`TransactionDate string` O.

`AccountInquiryCustomerTopUpResponse`: `ResponseCode string`,
`ResponseMessage string`, `ReferenceNo string` O, `PartnerReferenceNo
string` O, `SessionID string` O, `CustomerNumber string` C (omitempty)
+ `CustomerName string` M (no omitempty) + `CustomerMonthlyInLimit
json.RawMessage` O + `MinAmount *Money` O + `MaxAmount *Money` O +
`Amount *Money` O + `FeeAmount *Money` O + `FeeType string` O.

### CustomerTopUp (38)

`CustomerTopUpRequest`: `PartnerReferenceNo string` M (no omitempty) +
`CustomerNumber string` O + `CustomerName string` O + `Amount *Money`
O + `FeeAmount *Money` O + `TransactionDate string` O + `SessionID
string` O + `CategoryID json.RawMessage` O + `Notes string` O.

`CustomerTopUpResponse`: `ResponseCode string`, `ResponseMessage
string`, `ReferenceNo string` C (omitempty) + `PartnerReferenceNo
string` O + `SessionID string` O + `CustomerNumber string` O +
`Amount *Money` O + `ReferenceNumber string` O (undocumented in the
field table, present in the worked example — see above).

### CustomerTopUpInquiryStatus (39)

Field-identical to Phase 16's `TransactionStatusInquiryBankRequest`/
`Response` under new type names — see "Endpoint 39" above.

## Function behavior

All three follow the exact pattern already established across the
package: marshal `req`, set `hb.Body`, call `t.Do`,
`checkResponseStatus`, unmarshal into `Response`, error if
`ResponseCode == ""`. No method override — POST, matching every
endpoint in this sub-group.
