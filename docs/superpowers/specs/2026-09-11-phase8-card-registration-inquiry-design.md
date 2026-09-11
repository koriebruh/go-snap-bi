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
3. Parses `hb.EndpointURL` with `url.Parse` — rejecting anything that
   isn't a bare `http`/`https` URL with a host and no userinfo, query
   string, or fragment (`errInvalidEndpointURL`, checked via
   `errors.Is`) — rather than concatenating strings onto it. This went
   through three rounds. Round 2: string concatenation onto an
   `EndpointURL` ending in a fragment (`#...`) silently dropped
   `custIDMerchant` from the transmitted request (`net/http` never
   transmits a URL fragment), and one ending in a query string moved the
   segment into the query instead of the path — neither produced an
   error. That fix parsed the URL but still built the new path by hand
   (`u.Path = trimmed + "/custIdMerchant/" + custIDMerchant`) and cleared
   `u.RawPath` to force re-derivation from the decoded `Path` — which
   silently turned a `%2F`/`%2E%2E` already present in the *caller's
   own* `EndpointURL` into a real path boundary, reopening the exact
   traversal class step 2 closed, one field over. Round 3 fixed that
   with `u.JoinPath("custIdMerchant", custIDMerchant)`, but also caught
   an inaccurate description of *why* it works: JoinPath does not "leave
   RawPath alone" — it rebuilds the joined path from `u.EscapedPath()`
   and then re-derives both `Path` and `RawPath` from that
   already-escaped string (`net/url`'s `setPath`). A caller's existing
   encoding survives because it's carried through `EscapedPath()`, not
   because `RawPath` goes untouched — `RawPath` is in fact always
   overwritten. Round 3 also added the scheme/userinfo checks and a
   sentinel error, since the original `err != nil`-only tests for the
   `Opaque`/`Host` guards passed unchanged even with those guards
   deleted (the request still failed, just later, at the transport
   layer, for an unrelated reason).
4. Calls `t.Do` and decodes exactly like every other binding
   (`checkResponseStatus`, decode, reject empty `responseCode`).

This is a read-only GET with no side effects, so it needs no
idempotency note.

## Testing

13 tests: full-struct response DeepEqual (nested `accountList`), a
URL-construction test (asserting the exact request wire path via
`r.URL.EscapedPath()` for a valid `custIDMerchant`), a
rejected-`custIDMerchant` table test (`""`, `"."`, `".."`, a `/`-bearing
value, a multi-segment traversal attempt, a backslash variant, a
percent-encoded `..`, invalid UTF-8, a Unicode dot lookalike, and a
`?`-bearing value — all rejected before any request is sent, closing
the allowlist boundary review found gaps in), a `custIDMerchant`
length-boundary test (64 characters accepted, 65 rejected — the
allowlist's `{1,64}` bound was previously unpinned in either
direction), an invalid-`EndpointURL` table test asserting
`errors.Is(err, errInvalidEndpointURL)` for each of: a query string, a
fragment, a bare `?` (`ForceQuery` with an empty `RawQuery`), an opaque
URL, a non-http(s) scheme, a scheme-relative URL, and userinfo — plus a
separate empty-`EndpointURL` test, since `url.Parse("")` succeeds with
every field empty and so must be caught by the same `Host == ""` check
as an opaque URL rather than by a parse failure. Checking
`errors.Is` rather than only `err != nil` matters here: round 3 found
that the original version of this test suite passed unchanged even
with the `Opaque`/`Host` guards deleted, because the request still
failed downstream at the transport layer for an unrelated reason. Also:
a trailing-slash test (a caller's `EndpointURL` ending in `/` still
produces one, not two, slashes before `custIdMerchant`), an
encoded-path-preservation test (a caller's `EndpointURL` already
containing `%2F`/`%2E%2E` must reach the wire unchanged, not decoded and
reassembled), a signature-correctness test (recomputes
`BuildStringToSignTransaction` from the server-observed timestamp and
compares against the received `X-SIGNATURE`, closing the one part of
this GET-shaped request no other test exercises), non-2xx-responseCode,
non-2xx-status-with-2xx-body, 2xx-status-with-no-responseCode, and a
test asserting the request method is GET with no body regardless of
what the caller left on `hb.Body`/`hb.Method` before the call (proving
the override actually overrides caller-set values rather than only
filling in unset ones — verified via `r.ContentLength`, not a
best-effort one-byte `Read`).
