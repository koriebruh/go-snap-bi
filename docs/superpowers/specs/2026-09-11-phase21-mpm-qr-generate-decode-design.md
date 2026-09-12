# Phase 21: MPM / QR — Generate, Decode

Source: `docs/research/2026-09-11-transfer-kredit-portal-research.md` §5.9
(lines 190-194), Service Codes 47-48. Only these two endpoints — the
sub-group's other 6 (49-52, 77, 78) are deferred to Phase 22; §5.9's rows
for those are unread and batching them now would repeat Phase 18's
direction-assumption mistake.

## Open contradictions recorded, not resolved (research §6 item 8, plus one unflagged)

Decode QR MPM (48) response: `referenceNo` is documented "Mandatory if
redirect" and `redirectUrl` is documented "Mandatory if H2H mode" — these
read as describing opposite branches of the same conditional, which
research §6 item 8 flags as unresolved. Both fields are modeled Optional
(`omitempty`), matching the package's standard handling of every other
data-dependent Conditional field (e.g. Transfer To OTC Cancel Payment's
`cancelTime`, Phase 20) — the type system cannot express an either/or
branch condition, so neither reading is asserted.

Generate QR MPM (47) response: `qrContent` is Conditional with an explicit
rule — "if null, qrUrl or qrImage must be filled." A three-way
one-of-three condition across `qrContent`/`qrUrl`/`qrImage`, which Go's
type system likewise cannot express. All three are Optional
(`omitempty` strings); the rule is recorded in a doc comment, not
enforced.

## `merchantPAN` — ambiguous-type rule applied

`merchantInfos[].merchantPAN` is documented `Numeric(19)` but "quoted
string on wire" (research line 194) — the same shape as `customerNo`
(Phase 14/15), `channelCode`/`paymentType` (Phase 14/15), and
`CustomerMonthlyInLimit`/`CategoryID` (Phase 17): a documented
non-string-capable type that must round-trip a quoted value. Per the
package's standing ambiguous-type rule, this becomes `json.RawMessage`,
not `string` or a numeric Go type.

## `qrImage` — base64, unbounded

`qrImage` is `String(unlimited)`, base64-encoded. Modeled as a plain
`string` (no length enforcement, matching the package's general practice
of not validating field contents). This is the second place in the
package (after VA Get Report, Phase 15) where the shared 10 MiB transport
cap is the only practical bound on response size — noted here, not
re-implemented.

## Types

### GenerateQRMPM (47)

`GenerateQRMPMRequest`: `PartnerReferenceNo string` O + `Amount *Money` O
+ `FeeAmount *Money` O + `MerchantID string` O (String(64)) +
`SubMerchantID string` O (String(32)) + `StoreID string` O (String(64)) +
`TerminalID string` O (String(16)) + `ValidityPeriod string` O. Every
field in this request is Optional per §5.9 line 192 — no field is
promoted to mandatory in Go despite that being unusual for the package
(no prior request has had zero mandatory fields), because the research
row genuinely lists none as `M`.

`GenerateQRMPMResponse`: `ResponseCode string`, `ResponseMessage string`,
`QRContent string` C (omitempty; see contradiction note above) +
`QRUrl string` O (String(256)) + `QRImage string` O (base64,
unlimited) + `RedirectUrl string` O (String(512)) + `MerchantName string`
O + `StoreID string` O + `TerminalID string` O.

### DecodeQRMPM (48)

`DecodeQRMPMRequest`: `PartnerReferenceNo string` O + `QRContent string`
M (no omitempty) + `Amount *Money` O + `MerchantID string` O +
`SubMerchantID string` O + `ScanTime string` M (no omitempty;
String(25)).

`DecodeQRMPMResponse`: `ResponseCode string`, `ResponseMessage string`,
`ReferenceNo string` C (omitempty; see contradiction note) +
`RedirectUrl string` C (omitempty; see contradiction note) +
`MerchantName string` C + `MerchantCategory string` C +
`MerchantLocation string` C (all three from "merchantName/Category/
Location C") + `MerchantInfos []MPMMerchantInfo` M (no omitempty,
matching the package's mandatory-array convention — `BulkObject` in
Phase 12/18 is the precedent: mandatory arrays get no `omitempty`) +
`TransactionAmount *Money` O + `FeeAmount *Money` O.

`MPMMerchantInfo` (new shared type, one array-item shape, not reused
elsewhere): `MerchantPAN json.RawMessage` `json:"merchantPAN"` M (no
omitempty; ambiguous-type rule, see above) + `AcquirerName string`
`json:"acquirerName"` M (no omitempty; String(50)).

## Function behavior

Both follow the package's standard pattern: marshal `req`, set `hb.Body`,
call `t.Do`, `checkResponseStatus`, unmarshal into `Response`, error if
`ResponseCode == ""`. POST, no method override. `GenerateQRMPM` gets the
standard non-idempotency doc note (each call may mint a new QR);
`DecodeQRMPM` is a read-only decode and does not.

Paths from research §1's inventory table: `qr/qr-mpm-generate` (47),
`qr/qr-mpm-decode` (48).
