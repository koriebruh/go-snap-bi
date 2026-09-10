package snap

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
)

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
	if h.Get("Origin") != "" {
		t.Errorf("ORIGIN header = %q, want empty (not supplied)", h.Get("Origin"))
	}

	// B2B2C-only headers must be absent.
	for _, k := range []string{"Authorization-Customer", "X-Ip-Address", "X-Device-Id", "X-Latitude", "X-Longitude"} {
		if got := h.Get(k); got != "" {
			t.Errorf("B2B request: header %q = %q, want absent", k, got)
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

func TestHeaderBuilder_Build_B2B2C_OptionalFieldsOmittedWhenEmpty(t *testing.T) {
	b := HeaderBuilder{
		Method:       "POST",
		EndpointURL:  "https://openapi.example.com/v1.0/transfer-va",
		Body:         []byte(`{}`),
		B2B2C:        true, // B2B2C request, but none of the optional fields supplied
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

	for _, k := range []string{"Authorization-Customer", "X-Ip-Address", "X-Device-Id", "X-Latitude", "X-Longitude", "Origin"} {
		if got := h.Get(k); got != "" {
			t.Errorf("header %q = %q, want absent when not supplied", k, got)
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
