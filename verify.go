package snap

import (
	"crypto"
	"errors"
	"fmt"
	"time"
)

// KeyStore is caller-implemented — key/secret storage is an application
// concern (DB, vault, etc.), not something this package should own.
type KeyStore interface {
	PublicKey(clientKey string) (crypto.PublicKey, error)
	ClientSecret(clientKey string) (string, error)
}

// SignatureMode is the transaction-level signing mode agreed with a partner
// at registration time, per the standard's own wording — this is config,
// not a per-request runtime choice.
type SignatureMode int

const (
	SignatureModeSymmetric SignatureMode = iota // default zero value
	SignatureModeAsymmetric
)

// IncomingRequest holds the fields a ServerVerifier needs, extracted by the
// caller from whatever HTTP framework they use — this package doesn't parse
// *http.Request itself, to stay framework-agnostic.
type IncomingRequest struct {
	Method      string // HTTP method, e.g. "POST"
	EndpointURL string // full endpoint URL exactly as used in the signing formula
	Body        []byte // exact request body bytes as received
	Timestamp   string // X-TIMESTAMP header value
	ClientKey   string // X-CLIENT-KEY header value
	Signature   string // X-SIGNATURE header value
	AccessToken string // Authorization header's token, without the "Bearer " prefix; required for symmetric transaction verification, ignored otherwise
}

// ErrSignatureMismatch is returned when a signature fails to verify against
// an otherwise-successfully-looked-up key/secret, for both signing modes —
// giving callers one mode-independent errors.Is check rather than needing to
// know that VerifyAsymmetric happens to return its own error on mismatch.
var ErrSignatureMismatch = errors.New("snap: signature mismatch")

// ErrNoKeyStore is returned when a ServerVerifier is used with a nil
// KeyStore, instead of panicking on the first lookup.
var ErrNoKeyStore = errors.New("snap: verify: KeyStore is not set")

// ServerVerifier verifies incoming SNAP requests on the Penyedia Layanan
// (server) side — the inverse of the client-side signing in HeaderBuilder
// and TokenManager.
type ServerVerifier struct {
	KeyStore        KeyStore
	Mode            SignatureMode    // transaction-level mode; access-token requests are always asymmetric regardless of this
	TimestampWindow time.Duration    // freshness tolerance; zero means no freshness check (explicit opt-in, not hardcoded)
	Profile         Profile          // optional; nil means DefaultProfile{}, used only for TimestampLayout when parsing Timestamp
	Now             func() time.Time // optional; nil means time.Now, injectable for tests
}

func (v *ServerVerifier) now() time.Time {
	if v.Now != nil {
		return v.Now()
	}
	return time.Now()
}

func (v *ServerVerifier) profile() Profile {
	if v.Profile != nil {
		return v.Profile
	}
	return DefaultProfile{}
}

// checkFreshness verifies req.Timestamp parses under the active Profile's
// TimestampLayout and falls within TimestampWindow of now, in either
// direction. A zero TimestampWindow is an explicit opt-out — the timestamp
// is not even parsed in that case.
func (v *ServerVerifier) checkFreshness(timestamp string) error {
	if v.TimestampWindow <= 0 {
		return nil
	}
	parsed, err := time.Parse(v.profile().TimestampLayout(), timestamp)
	if err != nil {
		return fmt.Errorf("snap: verify: parse timestamp: %w", err)
	}
	skew := v.now().Sub(parsed)
	if skew < 0 {
		skew = -skew
	}
	if skew > v.TimestampWindow {
		return fmt.Errorf("snap: verify: timestamp %s outside freshness window %s", truncateForError(timestamp), v.TimestampWindow)
	}
	return nil
}

// VerifyAccessTokenRequest verifies an incoming access-token request's
// signature. Access-token requests are always asymmetric, regardless of
// v.Mode, matching TokenManager which always signs them asymmetrically.
func (v *ServerVerifier) VerifyAccessTokenRequest(req IncomingRequest) error {
	if v.KeyStore == nil {
		return ErrNoKeyStore
	}
	if err := v.checkFreshness(req.Timestamp); err != nil {
		return err
	}
	pub, err := v.KeyStore.PublicKey(req.ClientKey)
	if err != nil {
		return fmt.Errorf("snap: verify access token request: key lookup: %w", err)
	}
	stringToSign := BuildStringToSignAccessToken(req.ClientKey, req.Timestamp)
	if err := VerifyAsymmetric(pub, stringToSign, req.Signature); err != nil {
		return fmt.Errorf("snap: verify access token request: %w: %w", ErrSignatureMismatch, err)
	}
	return nil
}

// VerifyTransactionRequest verifies an incoming transaction request's
// signature per the pre-agreed SignatureMode. A KeyStore lookup succeeding
// does not by itself make the request valid — the signature check always
// runs and is what determines pass/fail.
func (v *ServerVerifier) VerifyTransactionRequest(req IncomingRequest) error {
	if v.KeyStore == nil {
		return ErrNoKeyStore
	}
	if err := v.checkFreshness(req.Timestamp); err != nil {
		return err
	}

	symmetric := v.Mode == SignatureModeSymmetric
	stringToSign := BuildStringToSignTransaction(req.Method, req.EndpointURL, req.AccessToken, req.Body, req.Timestamp, symmetric)

	if symmetric {
		secret, err := v.KeyStore.ClientSecret(req.ClientKey)
		if err != nil {
			return fmt.Errorf("snap: verify transaction request: secret lookup: %w", err)
		}
		if !VerifySymmetric(secret, stringToSign, req.Signature) {
			return fmt.Errorf("snap: verify transaction request: %w", ErrSignatureMismatch)
		}
		return nil
	}

	pub, err := v.KeyStore.PublicKey(req.ClientKey)
	if err != nil {
		return fmt.Errorf("snap: verify transaction request: key lookup: %w", err)
	}
	if err := VerifyAsymmetric(pub, stringToSign, req.Signature); err != nil {
		return fmt.Errorf("snap: verify transaction request: %w: %w", ErrSignatureMismatch, err)
	}
	return nil
}
