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
length 18), appended as the final path segment. This package validates
it against `^[A-Za-z0-9_-]{1,64}$` (see Design) rather than passing any
string through — narrower than the raw "String" the Guides tab allows,
but every worked example in the standard is alphanumeric, and no
endpoint in this package accepts a business identifier containing `.`,
`/`, or non-ASCII characters.

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
2. Validates `custIDMerchant` against `custIDMerchantPattern`
   (`^[A-Za-z0-9_-]{1,64}$`) — an allowlist, not a denylist. This is the
   package's first caller input that reaches a URL path rather than a
   JSON body, which is a different trust boundary than every other
   endpoint's fields: the package's `BalanceInquiryRequest`/
   `AccountUnbindingRequest` precedent for skipping client-side
   validation applies to JSON *body* fields, where a bad value can only
   be rejected by the server; it doesn't transfer to a URL *path*
   segment, where a bad value can retarget the request to a different
   resource before the server ever validates anything.

   This went through two review rounds before landing on an allowlist.
   `url.PathEscape` does escape `/` but not `.`/`..` (both are ordinary
   path-segment characters; only their special two-dot-meaning is
   dangerous, and escaping can't tell literal dots from traversal dots),
   so a first fix denylisted `""`, `"."`, `".."`. Santa-loop review then
   found the denylist incomplete: a multi-segment value like
   `"x/../../other"` still passes it and reaches the wire as one escaped
   segment (`"x%2F..%2F..%2Fother"`), which a decode-then-normalize
   intermediary could unfold back into real path boundaries — and
   invalid UTF-8 or Unicode dot lookalikes (e.g. U+2024) aren't covered
   by a three-value list at all. An allowlist restricted to the
   character set every worked identifier in the standard actually uses
   closes the whole class in one guard instead of enumerating variants.
3. Parses `hb.EndpointURL` with `url.Parse` — rejecting one with a
   non-empty query string or fragment — rather than concatenating
   strings onto it. This is also a santa-loop finding: string
   concatenation onto an `EndpointURL` ending in a fragment (`#...`)
   silently dropped `custIDMerchant` from the transmitted request
   (`net/http` never transmits a URL fragment), and one ending in a
   query string moved the segment into the query instead of the path —
   neither produced an error. Trims any trailing `/` from the parsed
   path (not the whole string) before appending
   `"/custIdMerchant/" + custIDMerchant`, so a caller's `EndpointURL`
   ending in any number of slashes still produces exactly one.
4. Calls `t.Do` and decodes exactly like every other binding
   (`checkResponseStatus`, decode, reject empty `responseCode`).

This is a read-only GET with no side effects, so it needs no
idempotency note.

## Testing

10 tests: full-struct response DeepEqual (nested `accountList`), a
URL-construction test (asserting the exact request wire path via
`r.URL.EscapedPath()` for a valid `custIDMerchant`), a
rejected-`custIDMerchant` table test (`""`, `"."`, `".."`, a `/`-bearing
value, a multi-segment traversal attempt, a backslash variant, a
percent-encoded `..`, invalid UTF-8, a Unicode dot lookalike, and a
`?`-bearing value — all rejected before any request is sent, closing
the allowlist boundary both review rounds found gaps in), a
query-or-fragment-`EndpointURL`-rejection test, a trailing-slash test
(a caller's `EndpointURL` ending in `/` still produces one, not two,
slashes before `custIdMerchant`), a signature-correctness test
(recomputes `BuildStringToSignTransaction` from the server-observed
timestamp and compares against the received `X-SIGNATURE`, closing the
one part of this GET-shaped request no other test exercises),
non-2xx-responseCode, non-2xx-status-with-2xx-body,
2xx-status-with-no-responseCode, and a test asserting the request
method is GET with no body regardless of what the caller left on
`hb.Body`/`hb.Method` before the call (proving the override actually
overrides caller-set values rather than only filling in unset ones —
verified via `r.ContentLength`, not a best-effort one-byte `Read`).
