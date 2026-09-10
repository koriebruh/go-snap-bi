package snap

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"testing"
)

// headerAbsent reports whether k is genuinely not present in h, as opposed
// to present with an empty value — h.Get(k) == "" can't tell those apart,
// which would let a header wrongly set to "" pass an "omitted" assertion.
func headerAbsent(h http.Header, k string) bool {
	_, ok := h[http.CanonicalHeaderKey(k)]
	return !ok
}

// fixedLayoutProfile overrides TimestampLayout to prove the Profile hook
// actually takes effect on the built X-TIMESTAMP header.
type fixedLayoutProfile struct {
	DefaultProfile
}

func (fixedLayoutProfile) TimestampLayout() string {
	return "2006" // deliberately distinguishable from DefaultTimestampLayout
}

func TestHeaderBuilder_Build_B2B(t *testing.T) {
	b := HeaderBuilder{
		Method:       "POST",
		EndpointURL:  "https://openapi.example.com/v1.0/transfer-va",
		Body:         []byte(`{"amount":"1000"}`),
		B2B2C:        false,
		AccessToken:  "access-token-123",
		ClientKey:    "client-key",
		PartnerID:    "partner-id",
		ExternalID:   "external-id",
		ChannelID:    "channel-id",
		Symmetric:    true,
		ClientSecret: "shh-secret",
	}

	h, err := b.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	wantAlways := map[string]string{
		"Content-Type":  "application/json",
		"X-Client-Key":  "client-key",
		"X-Partner-Id":  "partner-id",
		"X-External-Id": "external-id",
		"Channel-Id":    "channel-id",
		"Authorization": "Bearer access-token-123",
	}
	for k, want := range wantAlways {
		if got := h.Get(k); got != want {
			t.Errorf("header %q = %q, want %q", k, got, want)
		}
	}
	if h.Get("X-Timestamp") == "" {
		t.Error("X-TIMESTAMP header is empty")
	}
	if h.Get("X-Signature") == "" {
		t.Error("X-SIGNATURE header is empty")
	}
	if !headerAbsent(h, "Origin") {
		t.Errorf("ORIGIN header present = %q, want absent (not supplied)", h.Get("Origin"))
	}

	// B2B2C-only headers must be absent.
	for _, k := range []string{"Authorization-Customer", "X-Ip-Address", "X-Device-Id", "X-Latitude", "X-Longitude"} {
		if !headerAbsent(h, k) {
			t.Errorf("B2B request: header %q present = %q, want absent", k, h.Get(k))
		}
	}

	// The signature must genuinely verify against the recomputed
	// stringToSign, using the timestamp actually placed in the header, and
	// must fail against the opposite (asymmetric) formula, proving the
	// builder threads the Symmetric flag through rather than hardcoding one
	// signing path.
	ts := h.Get("X-Timestamp")
	sts := BuildStringToSignTransaction(b.Method, b.EndpointURL, b.AccessToken, b.Body, ts, true)
	if !VerifySymmetric(b.ClientSecret, sts, h.Get("X-Signature")) {
		t.Error("X-SIGNATURE does not verify against the symmetric stringToSign")
	}
	stsAsymmetric := BuildStringToSignTransaction(b.Method, b.EndpointURL, b.AccessToken, b.Body, ts, false)
	if VerifySymmetric(b.ClientSecret, stsAsymmetric, h.Get("X-Signature")) {
		t.Error("X-SIGNATURE unexpectedly verifies against the asymmetric stringToSign")
	}
}

func TestHeaderBuilder_Build_Asymmetric(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}

	b := HeaderBuilder{
		Method:      "POST",
		EndpointURL: "https://openapi.example.com/v1.0/transfer-va",
		Body:        []byte(`{"amount":"1000"}`),
		AccessToken: "access-token-123",
		ClientKey:   "client-key",
		PartnerID:   "partner-id",
		ExternalID:  "external-id",
		ChannelID:   "channel-id",
		Symmetric:   false,
		Signer:      key,
	}

	h, err := b.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	ts := h.Get("X-Timestamp")
	sts := BuildStringToSignTransaction(b.Method, b.EndpointURL, b.AccessToken, b.Body, ts, false)
	if err := VerifyAsymmetric(&key.PublicKey, sts, h.Get("X-Signature")); err != nil {
		t.Errorf("X-SIGNATURE does not verify against the asymmetric stringToSign: %v", err)
	}
}

func TestHeaderBuilder_Build_B2B2C(t *testing.T) {
	b := HeaderBuilder{
		Method:                "POST",
		EndpointURL:           "https://openapi.example.com/v1.0/transfer-va",
		Body:                  []byte(`{"amount":"1000"}`),
		B2B2C:                 true,
		AccessToken:           "access-token-123",
		AuthorizationCustomer: "customer-token-456",
		ClientKey:             "client-key",
		PartnerID:             "partner-id",
		ExternalID:            "external-id",
		ChannelID:             "channel-id",
		Origin:                "https://partner.example.com",
		IPAddress:             "10.0.0.1",
		DeviceID:              "device-1",
		Latitude:              "-6.2",
		Longitude:             "106.8",
		Symmetric:             true,
		ClientSecret:          "shh-secret",
	}

	h, err := b.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	want := map[string]string{
		"Authorization-Customer": "Bearer customer-token-456",
		"X-Ip-Address":           "10.0.0.1",
		"X-Device-Id":            "device-1",
		"X-Latitude":             "-6.2",
		"X-Longitude":            "106.8",
		"Origin":                 "https://partner.example.com",
	}
	for k, wantVal := range want {
		if got := h.Get(k); got != wantVal {
			t.Errorf("B2B2C header %q = %q, want %q", k, got, wantVal)
		}
	}
}

// TestHeaderBuilder_Build_B2B2C_GenuinelyOptionalFieldsOmittedWhenEmpty
// covers only IPAddress/Latitude/Longitude/Origin, which the standard marks
// optional. AuthorizationCustomer and X-DEVICE-ID are mandatory for B2B2C
// per the standard (Pedoman Standar Teknis dan Keamanan SNAP, §2.1.6.b) and
// are covered by the mandatory-field error tests below instead.
func TestHeaderBuilder_Build_B2B2C_GenuinelyOptionalFieldsOmittedWhenEmpty(t *testing.T) {
	b := HeaderBuilder{
		Method:                "POST",
		EndpointURL:           "https://openapi.example.com/v1.0/transfer-va",
		Body:                  []byte(`{}`),
		B2B2C:                 true,
		AccessToken:           "access-token-123",
		AuthorizationCustomer: "customer-token-456", // mandatory, supplied
		DeviceID:              "device-1",           // mandatory, supplied
		ClientKey:             "client-key",
		PartnerID:             "partner-id",
		ExternalID:            "external-id",
		ChannelID:             "channel-id",
		Symmetric:             true,
		ClientSecret:          "shh-secret",
	}

	h, err := b.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	for _, k := range []string{"X-Ip-Address", "X-Latitude", "X-Longitude", "Origin"} {
		if !headerAbsent(h, k) {
			t.Errorf("header %q present = %q, want absent when not supplied", k, h.Get(k))
		}
	}
}

func TestHeaderBuilder_Build_MandatoryFieldErrors(t *testing.T) {
	base := func() HeaderBuilder {
		return HeaderBuilder{
			Method:       "POST",
			EndpointURL:  "https://openapi.example.com/v1.0/transfer-va",
			Body:         []byte(`{}`),
			AccessToken:  "access-token-123",
			ClientKey:    "client-key",
			PartnerID:    "partner-id",
			ExternalID:   "external-id",
			ChannelID:    "channel-id",
			Symmetric:    true,
			ClientSecret: "shh-secret",
		}
	}

	t.Run("missing ExternalID", func(t *testing.T) {
		b := base()
		b.ExternalID = ""
		if _, err := b.Build(); err == nil {
			t.Error("Build() with empty ExternalID: want error, got nil")
		}
	})

	t.Run("symmetric with empty ClientSecret", func(t *testing.T) {
		b := base()
		b.ClientSecret = ""
		if _, err := b.Build(); err == nil {
			t.Error("Build() with Symmetric=true and empty ClientSecret: want error, got nil")
		}
	})

	t.Run("asymmetric with nil Signer", func(t *testing.T) {
		b := base()
		b.Symmetric = false
		b.ClientSecret = ""
		b.Signer = nil
		if _, err := b.Build(); err == nil {
			t.Error("Build() with Symmetric=false and nil Signer: want error, got nil")
		}
	})

	t.Run("B2B2C missing AuthorizationCustomer", func(t *testing.T) {
		b := base()
		b.B2B2C = true
		b.DeviceID = "device-1"
		if _, err := b.Build(); err == nil {
			t.Error("Build() with B2B2C=true and empty AuthorizationCustomer: want error, got nil")
		}
	})

	t.Run("B2B2C missing DeviceID", func(t *testing.T) {
		b := base()
		b.B2B2C = true
		b.AuthorizationCustomer = "customer-token-456"
		if _, err := b.Build(); err == nil {
			t.Error("Build() with B2B2C=true and empty DeviceID: want error, got nil")
		}
	})
}

// TestHeaderBuilder_Build_B2BDoesNotLeakB2B2CFields is a regression guard: a
// B2B request (B2B2C: false) must never emit customer-scoped headers even if
// the corresponding struct fields happen to be populated by the caller (e.g.
// a reused config struct), since a future refactor moving one h.Set outside
// the `if b.B2B2C` block would otherwise pass every other existing test.
func TestHeaderBuilder_Build_B2BDoesNotLeakB2B2CFields(t *testing.T) {
	b := HeaderBuilder{
		Method:                "POST",
		EndpointURL:           "https://openapi.example.com/v1.0/transfer-va",
		Body:                  []byte(`{}`),
		B2B2C:                 false,
		AccessToken:           "access-token-123",
		AuthorizationCustomer: "customer-token-456",
		DeviceID:              "device-1",
		IPAddress:             "10.0.0.1",
		Latitude:              "-6.2",
		Longitude:             "106.8",
		ClientKey:             "client-key",
		PartnerID:             "partner-id",
		ExternalID:            "external-id",
		ChannelID:             "channel-id",
		Symmetric:             true,
		ClientSecret:          "shh-secret",
	}

	h, err := b.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	for _, k := range []string{"Authorization-Customer", "X-Device-Id", "X-Ip-Address", "X-Latitude", "X-Longitude"} {
		if !headerAbsent(h, k) {
			t.Errorf("B2B request: header %q present = %q, want absent even though the field was populated", k, h.Get(k))
		}
	}
}

func TestHeaderBuilder_Build_CustomProfileChangesTimestamp(t *testing.T) {
	base := HeaderBuilder{
		Method:       "POST",
		EndpointURL:  "https://openapi.example.com/v1.0/transfer-va",
		Body:         []byte(`{}`),
		AccessToken:  "access-token-123",
		ClientKey:    "client-key",
		PartnerID:    "partner-id",
		ExternalID:   "external-id",
		ChannelID:    "channel-id",
		Symmetric:    true,
		ClientSecret: "shh-secret",
	}

	withDefault := base
	withDefault.Profile = DefaultProfile{}
	hDefault, err := withDefault.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	withCustom := base
	withCustom.Profile = fixedLayoutProfile{}
	hCustom, err := withCustom.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	tsDefault := hDefault.Get("X-Timestamp")
	tsCustom := hCustom.Get("X-Timestamp")

	if len(tsDefault) == len(tsCustom) {
		t.Fatalf("expected custom Profile's TimestampLayout to change X-TIMESTAMP's format, got same length %q vs %q", tsDefault, tsCustom)
	}
	if len(tsCustom) != 4 {
		t.Errorf("custom X-TIMESTAMP = %q, want a 4-digit year per the stub layout", tsCustom)
	}
}
