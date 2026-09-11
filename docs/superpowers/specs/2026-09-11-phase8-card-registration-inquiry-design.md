# go-snap-bi: Phase 8 — Card Registration Inquiry (Registrasi)

Status: implemented. Scope: one endpoint, isolated into its own phase
because its shape is new for this package: GET, no request body, a path
parameter instead of a JSON field, and a nested array response.

## API Card Registration Inquiry (Service Code 03)

Path `.../{version}/registration-card-inquiry/custIdMerchant/{value}`
(`custIdMerchant` is a URL path segment, not a query parameter or body
field). GET. Confirmed via worked `responseCode` `2000300`.

### Request

No JSON body (GET). One input: `custIdMerchant` (String, Mandatory,
length 18), appended as the final path segment.

### Response body

| Field | Guides tab type | M/O | Go type |
|---|---|---|---|
| responseCode | String | M | `string` |
| responseMessage | String | M | `string` |
| accountList | Array of Objects | — | `[]CardRegistrationInquiryAccount` |
| accountList[].accountData.accountId | String | O | `string` |
| accountList[].accountData.createdDate | String | O | `string` |
| accountList[].accountData.credentialNo | String | O | `string` |
| accountList[].accountData.credentialType | String | O | `string` |
| accountList[].accountData.maxLimit | String | O | `string` |
| accountList[].accountData.status | String | O | `string` |
| additionalInfo | Object | O | `json.RawMessage` |

`maxLimit` reads number-like (worked example `"800000"`) but the Guides
tab labels it String, and the value is a masked/formatted display string
(the response's `credentialNo` is masked the same way, e.g.
`"************0750"`) — not the raw numeric limit `CardRegistrationSetLimitRequest.Limit`
sets. `string` is unambiguous here, unlike Phase 7's `limit`/`cardData`.

## Design

New file `card_registration_inquiry.go`. Types:
`CardRegistrationInquiryAccountData` (the nested `accountData` object),
`CardRegistrationInquiryAccount` (one `accountList` entry — a struct
wrapping `AccountData` to mirror the wire shape's extra nesting level
exactly, not flattened), `CardRegistrationInquiryResponse`.

`CardRegistrationInquiry(ctx, t, hb, custIDMerchant string) (CardRegistrationInquiryResponse, error)`
takes `custIDMerchant` as a plain parameter, not a request struct — there
is no JSON body for a GET, so a struct would only hold the one path value
with nowhere to put a JSON tag. The function:

1. Sets `hb.Method = http.MethodGet` and `hb.Body = nil` itself, rather
   than relying on the caller to configure them correctly — this is a
   protocol requirement of this specific endpoint, not something a caller
   should choose per call (every other implemented endpoint is POST, so
   there's no existing caller expectation to preserve either way).
2. Rejects `custIDMerchant` values `""`, `"."`, and `".."` outright, then
   appends the value to `hb.EndpointURL` (with any trailing `/` trimmed
   first) via `url.PathEscape`, not raw string concatenation. This is the
   package's first caller input that reaches a URL path rather than a
   JSON body, and the two hazards are different: `url.PathEscape` DOES
   escape `/` (santa-loop review confirmed: a `custIDMerchant` containing
   `/` cannot add an extra path segment), but it does NOT escape `.` —
   `.`/`..` are ordinary path-segment characters that only become
   dangerous through their special filesystem-style meaning, which
   escaping can't distinguish from a literal dot. That's a go-review
   finding on this phase: the package's own `BalanceInquiryRequest`/
   `AccountUnbindingRequest` precedent for skipping client-side
   validation applies to JSON *body* fields, where a bad value can only
   be rejected by the server — it doesn't transfer to a URL *path*
   segment, where a bad value can silently redirect the request to a
   different resource (a `..` walks up one path level) before the
   server ever sees it.
3. Calls `t.Do` and decodes exactly like every other binding
   (`checkResponseStatus`, decode, reject empty `responseCode`).

This is a read-only GET with no side effects, so it needs no
idempotency note.

## Testing

9 tests: full-struct response DeepEqual (nested `accountList`), a
URL-construction test (asserting the exact request wire path via
`r.URL.EscapedPath()`, including escaping a `custIDMerchant` value
containing a `/` — `r.URL.Path` decodes `%2F` back to `/`, so it can't
tell an escaped slash from a real path boundary), a
path-traversal-rejection test (`""`, `"."`, `".."` all rejected before
any request is sent), a trailing-slash test (a caller's `EndpointURL`
ending in `/` still produces one, not two, slashes before
`custIdMerchant`), a signature-correctness test (recomputes
`BuildStringToSignTransaction` from the server-observed timestamp and
compares against the received `X-SIGNATURE`, closing the one part of
this GET-shaped request no other test exercises), non-2xx-responseCode,
non-2xx-status-with-2xx-body, 2xx-status-with-no-responseCode, and a
test asserting the request method is GET with no body regardless of
what the caller left on `hb.Body`/`hb.Method` before the call (proving
the override actually overrides caller-set values rather than only
filling in unset ones — verified via `r.ContentLength`, not a
best-effort one-byte `Read`).
