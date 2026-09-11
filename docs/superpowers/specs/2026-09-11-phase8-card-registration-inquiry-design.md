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
2. Appends `custIDMerchant` to `hb.EndpointURL` via `url.PathEscape`,
   not raw string concatenation — this is the package's first caller
   input that reaches a URL path rather than a JSON body, so it is also
   the first place a malformed or adversarial `custIDMerchant` value
   (e.g. containing `/` or `..`) could otherwise alter the request path
   instead of just being rejected as an invalid identifier value.
3. Calls `t.Do` and decodes exactly like every other binding
   (`checkResponseStatus`, decode, reject empty `responseCode`).

No client-side validation of `custIDMerchant` being non-empty: an empty
value produces a syntactically valid (if pointless) URL ending in
`.../custIdMerchant/`, and the server rejects it — same "the server
validates business rules the wire shape doesn't capture" stance as
`BalanceInquiryRequest`/`AccountUnbindingRequest`.

This is a read-only GET with no side effects, so it needs no
idempotency note.

## Testing

6 tests: full-struct response DeepEqual (nested `accountList`), a
URL-construction test (asserting the exact request path the server
received, including escaping a `custIDMerchant` value containing a `/`),
non-2xx-responseCode, non-2xx-status-with-2xx-body,
2xx-status-with-no-responseCode, and a test asserting the request method
is GET with no body regardless of what the caller left on `hb.Body`/`hb.Method`
before the call (proving step 1 above actually overrides caller-set
values rather than only filling in unset ones).
