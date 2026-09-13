// Package snaptest provides shared test-only helpers for go-snap-bi's
// transferkredit and transferdebit subpackages. It lives under internal/
// so it is importable anywhere in this module but not part of the
// public API surface.
package snaptest

import (
	"net/http"

	snap "github.com/koriebruh/go-snap-bi"
)

// TestHeaderBuilder returns a HeaderBuilder with placeholder credentials
// pointed at serverURL, for tests that stand up an httptest.Server.
// Most call sites override EndpointURL to the specific path under
// test; a few (e.g. BalanceInquiry's own tests) don't and get this
// placeholder path as-is, which is harmless since none of those
// handlers assert anything about the request path.
//
// Moved here from the single per-package copy each endpoint test file
// used to declare, so transferkredit and transferdebit share one
// definition instead of duplicating it.
func TestHeaderBuilder(serverURL string) snap.HeaderBuilder {
	return snap.HeaderBuilder{
		Method:       http.MethodPost,
		EndpointURL:  serverURL + "/v1.0/balance-inquiry",
		AccessToken:  "access-token-123",
		ClientKey:    "client-key",
		PartnerID:    "partner-id",
		ExternalID:   "external-id",
		ChannelID:    "channel-id",
		Symmetric:    true,
		ClientSecret: "shh-secret",
	}
}
