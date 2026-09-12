# Phase 22: MPM / QR — Payment Redirect Apply OTT, Payment Host-to-Host

Source: `docs/research/2026-09-11-transfer-kredit-portal-research.md` §5.9
(lines 196, 198), Service Codes 49-50. Only these two of the sub-group's
remaining 6 endpoints — 51/52 and 77/78 are deferred to Phase 23 and
Phase 24 respectively (per advisor guidance: 6 endpoints in one review
diluted reviewer attention across too many tags in Phase 21's
precedent).

## Endpoint 49 (Apply OTT) — request body is a bare JSON array, not an object

Research line 196: "Req: `userResources Array of String(64) M`. ...
Worked example request body is `["OTT"]`." This is the one anomaly in
this phase and the reason it was paired separately from 51/52/77/78.

The worked example shows a **top-level JSON array** as the request
body, not an object with a `userResources` key wrapping it — unlike
every other endpoint in the whole 43+-endpoint inventory, which sends
identifiers inside a JSON object body (research §1, "Architecturally
important" note). This is a genuine, unresolved contradiction: the
field table names a field `userResources`, but the worked example shows
no such key, just the bare array value.

Per the package's established practice for unresolved contradictions
(VA Get Report's GET/POST, Phase 15; Transfer To OTC Cancel's two
paths, Phase 20), the literal worked example is taken as authoritative
and the field-table reading is recorded as the alternative, not
implemented. `ApplyOTTRequest` is therefore a defined slice type
(`type ApplyOTTRequest []string`), marshaled directly as a top-level
array — not a struct with a `UserResources []string` field, which
would produce `{"userResources":[...]}` and contradict the worked
example. This needs sandbox/Postman verification before either reading
is treated as settled.

The response is a normal envelope object (no such contradiction is
raised for it): `userResources[] M (resourceType String(32) M, value
String(64) M)` — a mandatory array of a new item type, distinct from
the request's bare-string array despite the shared field name.

## Endpoint 50 (Payment Host-to-Host) — response base pattern is an interpretive call

Research line 198: "Req: `partnerReferenceNo M`, `merchantId/
subMerchantId O`, `amount/feeAmount O`, `otp String(8) O`,
`verificationId String(32) O`. Resp adds `verificationId String(64)
O`." Unlike endpoint 45's "originalX/serviceCode/status pattern" or
endpoint 51's "originalX/serviceCode pattern" (both of which name an
explicit base shape reused elsewhere in the document), line 198 does
not name what the response is adding *to*. This is a research
ambiguity, not a package convention decision.

Interpreted here as extending the package's minimal Payment-response
shape already used by sibling Payment endpoints (`TransferToOTCCreatePaymentResponse`,
Phase 20; the `referenceNo`/`transactionDate` core of Payment
Transaction, §5.7 line 180): `ResponseCode`, `ResponseMessage`,
`ReferenceNo` C, `TransactionDate` O, plus the explicitly stated
`VerificationID` O. This is a modeling choice under ambiguity, not a
literal transcription — recorded here so a future reader with sandbox
access can correct it if wrong.

`VerificationID` appears on both request (`String(32)`) and response
(`String(64)`) under the same field name with different documented
lengths. Both are plain `string` in Go (no length enforcement,
matching package practice) — this is not a type-shape difference, just
a length note worth recording so it isn't mistaken for a typo in a
future edit.

## Types

### ApplyOTT (49)

`ApplyOTTRequest`: `type ApplyOTTRequest []string` — marshaled as a
bare JSON array. See contradiction note above.

`ApplyOTTUserResource` (new item type, response-only): `ResourceType
string` `json:"resourceType"` M (no omitempty; String(32)) + `Value
string` `json:"value"` M (no omitempty; String(64)).

`ApplyOTTResponse`: `ResponseCode string`, `ResponseMessage string`,
`UserResources []ApplyOTTUserResource` `json:"userResources"` M (no
omitempty, matching the package's mandatory-array convention —
`BulkObject`/`MerchantInfos` precedent).

### QRMPMPaymentH2H (50)

`QRMPMPaymentH2HRequest`: `PartnerReferenceNo string` M (no omitempty)
+ `MerchantID string` O + `SubMerchantID string` O + `Amount *Money` O
+ `FeeAmount *Money` O + `OTP string` O (String(8)) + `VerificationID
string` O (String(32)).

`QRMPMPaymentH2HResponse`: `ResponseCode string`, `ResponseMessage
string`, `ReferenceNo string` C (omitempty) + `TransactionDate string`
O + `VerificationID string` O (String(64) — see length note above).

## Function behavior

Both follow the package's standard pattern: marshal `req`, set
`hb.Body`, call `t.Do`, `checkResponseStatus`, unmarshal into
`Response`, error if `ResponseCode == ""`. POST, no method override.
Both are mutating/state-changing calls and get the standard
non-idempotency doc note.

Paths from research §1's inventory table: `qr/apply-ott` (49, line
50 — not the `qr/qr-mpm-ott` slug a naming-pattern guess would produce),
`qr/qr-mpm-payment` (50, line 51).
