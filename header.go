package snap

import (
	"crypto"
	"fmt"
	"net/http"
	"time"
)

// HeaderBuilder assembles the mandatory SNAP header set for a transaction
// request. Symmetric selects which of Slice 1's signing functions is used,
// matching BuildStringToSignTransaction's own parameter, so the caller
// supplies exactly one of ClientSecret or Signer.
type HeaderBuilder struct {
	Method      string // HTTP method, e.g. "POST"
	EndpointURL string // full endpoint URL used in the signed string
	Body        []byte // exact request body bytes
	B2B2C       bool   // whether this is a B2B2C request

	AccessToken           string // Authorization bearer token
	AuthorizationCustomer string // Authorization-Customer bearer token (B2B2C only)

	ClientKey  string
	PartnerID  string
	ExternalID string
	ChannelID  string
	Origin     string // optional; header omitted when empty

	// B2B2C-only, optional; each header is omitted when its value is empty.
	IPAddress string
	DeviceID  string
	Latitude  string
	Longitude string

	Profile Profile

	Symmetric    bool
	ClientSecret string        // used when Symmetric is true
	Signer       crypto.Signer // used when Symmetric is false
}

// Build assembles the http.Header for this request, signing it via
// BuildStringToSignTransaction and SignSymmetric/SignAsymmetric per the
// Symmetric flag.
func (b HeaderBuilder) Build() (http.Header, error) {
	profile := b.Profile
	if profile == nil {
		profile = DefaultProfile{}
	}
	timestamp := time.Now().Format(profile.TimestampLayout())

	stringToSign := BuildStringToSignTransaction(b.Method, b.EndpointURL, b.AccessToken, b.Body, timestamp, b.Symmetric)

	var signature string
	if b.Symmetric {
		signature = SignSymmetric(b.ClientSecret, stringToSign)
	} else {
		var err error
		signature, err = SignAsymmetric(b.Signer, stringToSign)
		if err != nil {
			return nil, fmt.Errorf("snap: build headers: %w", err)
		}
	}

	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set("X-TIMESTAMP", timestamp)
	h.Set("X-CLIENT-KEY", b.ClientKey)
	h.Set("X-SIGNATURE", signature)
	h.Set("X-PARTNER-ID", b.PartnerID)
	h.Set("X-EXTERNAL-ID", b.ExternalID)
	h.Set("CHANNEL-ID", b.ChannelID)
	h.Set("Authorization", "Bearer "+b.AccessToken)

	if b.Origin != "" {
		h.Set("ORIGIN", b.Origin)
	}

	if b.B2B2C {
		if b.AuthorizationCustomer != "" {
			h.Set("Authorization-Customer", "Bearer "+b.AuthorizationCustomer)
		}
		if b.IPAddress != "" {
			h.Set("X-IP-ADDRESS", b.IPAddress)
		}
		if b.DeviceID != "" {
			h.Set("X-DEVICE-ID", b.DeviceID)
		}
		if b.Latitude != "" {
			h.Set("X-LATITUDE", b.Latitude)
		}
		if b.Longitude != "" {
			h.Set("X-LONGITUDE", b.Longitude)
		}
	}

	return h, nil
}
