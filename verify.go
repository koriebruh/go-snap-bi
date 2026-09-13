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
// not a per-request runtime choice. The zero value is
// SignatureModeAsymmetric to match HeaderBuilder.Symmetric's zero value
// (false, i.e. asymmetric) — the two are meant to mirror each other for the
// same partner configuration, on the client and server side respectively.
type SignatureMode int

// The two SignatureMode values a ServerVerifier can be configured with.
const (
	SignatureModeAsymmetric SignatureMode = iota // default zero value; matches HeaderBuilder{}'s default (Symmetric: false)
	SignatureModeSymmetric                       // matches HeaderBuilder{Symmetric: true}
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
	ExternalID  string // X-EXTERNAL-ID header value; carried through for a caller's own replay defense (see VerifyTransactionRequest's doc comment) but not checked by this package
}

// ErrSignatureMismatch is returned when a signature fails to verify against
// an otherwise-successfully-looked-up key/secret, for both signing modes —
// giving callers one mode-independent errors.Is check rather than needing to
// know that VerifyAsymmetric happens to return its own error on mismatch.
//
// This is deliberately NOT returned for a KeyStore/config mistake — an
// empty ClientSecret (see ErrEmptyClientSecret) or an unusable RSA key
// (ErrNotRSASigner, ErrWeakRSAKey, both from signing.go) are distinct,
// separately errors.Is-matchable failures, so an operator paging on
// ErrSignatureMismatch alone is not woken for what is actually a
// configuration bug rather than a tampered or forged request.
var ErrSignatureMismatch = errors.New("snap: signature mismatch")

// ErrEmptyClientSecret is returned when a KeyStore's ClientSecret lookup
// succeeds but returns an empty string. Deliberately distinct from
// ErrSignatureMismatch: VerifySymmetric already refuses to treat an empty
// secret as "verifies against the empty string" (an empty HMAC key is a
// publicly computable MAC), but that refusal is otherwise
// indistinguishable from a genuine tampered-signature failure to a
// caller checking errors.Is(err, ErrSignatureMismatch) — this sentinel
// lets a caller tell "your KeyStore has a bug" apart from "this request
// was tampered with or forged."
var ErrEmptyClientSecret = errors.New("snap: verify: KeyStore.ClientSecret returned an empty string")

// ErrNoKeyStore is returned when a ServerVerifier is used with a nil
// KeyStore, instead of panicking on the first lookup.
var ErrNoKeyStore = errors.New("snap: verify: KeyStore is not set")

// DefaultTimestampWindow is the freshness tolerance used when
// ServerVerifier.TimestampWindow is left at its zero value. The standard
// requires timestamp freshness checking, so an unconfigured ServerVerifier
// fails closed with this sane default rather than silently skipping the
// check — matching every other optional field in this package (Now,
// Profile, HTTPClient), where absent means "safe default," never "disabled."
const DefaultTimestampWindow = 5 * time.Minute

// DisableTimestampFreshnessCheck is a sentinel TimestampWindow value that
// explicitly disables freshness checking. Distinct from the zero value on
// purpose, so "I forgot to set this" (zero value, gets DefaultTimestampWindow)
// and "I deliberately don't want this" (this sentinel) can't be confused.
const DisableTimestampFreshnessCheck time.Duration = -1

// ServerVerifier verifies incoming SNAP requests on the Penyedia Layanan
// (server) side — the inverse of the client-side signing in HeaderBuilder
// and TokenManager.
type ServerVerifier struct {
	KeyStore        KeyStore
	Mode            SignatureMode    // transaction-level mode; access-token requests are always asymmetric regardless of this
	TimestampWindow time.Duration    // freshness tolerance; zero means DefaultTimestampWindow, DisableTimestampFreshnessCheck explicitly disables it
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
// direction. A zero TimestampWindow uses DefaultTimestampWindow; only the
// explicit DisableTimestampFreshnessCheck sentinel skips the check entirely.
func (v *ServerVerifier) checkFreshness(timestamp string) error {
	window := v.TimestampWindow
	if window == DisableTimestampFreshnessCheck {
		return nil
	}
	if window == 0 {
		window = DefaultTimestampWindow
	}
	parsed, err := time.Parse(v.profile().TimestampLayout(), timestamp)
	if err != nil {
		return fmt.Errorf("snap: verify: parse timestamp %s: invalid format", TruncateForError(timestamp))
	}
	skew := v.now().Sub(parsed)
	if skew < 0 {
		skew = -skew
	}
	if skew > window {
		return fmt.Errorf("snap: verify: timestamp %s outside freshness window %s", TruncateForError(timestamp), window)
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
		if errors.Is(err, ErrNotRSASigner) || errors.Is(err, ErrWeakRSAKey) {
			// A KeyStore/config problem (the public key isn't a usable RSA
			// key), not evidence the request was tampered with — don't
			// fold it under ErrSignatureMismatch.
			return fmt.Errorf("snap: verify access token request: %w", err)
		}
		return fmt.Errorf("snap: verify access token request: %w: %w", ErrSignatureMismatch, err)
	}
	return nil
}

// VerifyTransactionRequest verifies an incoming transaction request's
// signature per the pre-agreed SignatureMode. A KeyStore lookup succeeding
// does not by itself make the request valid — the signature check always
// runs and is what determines pass/fail.
//
// A nil return proves the caller possesses the shared secret or private key
// for req.ClientKey and that the signature is bound to req.AccessToken as
// given — it does not itself validate that req.AccessToken is genuine,
// unexpired, or was actually issued to this client. Token issuance and
// introspection are out of this package's scope; callers who need that
// check must perform it separately.
//
// This function also does not itself defend against replay: within
// TimestampWindow, a captured, otherwise-untampered request verifies
// identically on a second delivery. The standard requires X-EXTERNAL-ID
// uniqueness per client per day for exactly this reason — req.ExternalID
// is carried through so a caller can maintain their own dedup store (a
// cache, a database unique constraint) keyed on ClientKey+ExternalID,
// but this package does not do so itself, the same way it does not own
// KeyStore's storage.
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
		if secret == "" {
			// Same config-vs-tampering distinction as the asymmetric
			// branch below: an empty secret means the KeyStore itself is
			// misconfigured for this client, not that the request was
			// forged, so it must not be reported as ErrSignatureMismatch.
			return fmt.Errorf("snap: verify transaction request: %w", ErrEmptyClientSecret)
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
		if errors.Is(err, ErrNotRSASigner) || errors.Is(err, ErrWeakRSAKey) {
			return fmt.Errorf("snap: verify transaction request: %w", err)
		}
		return fmt.Errorf("snap: verify transaction request: %w: %w", ErrSignatureMismatch, err)
	}
	return nil
}
