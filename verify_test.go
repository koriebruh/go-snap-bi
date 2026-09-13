package snap

import (
	"crypto"
	"errors"
	"fmt"
	"testing"
	"time"
)

// errUnknownClientKey is fakeKeyStore's own sentinel for an unregistered
// clientKey, distinct from ErrSignatureMismatch, so lookup-failure tests can
// prove the two are never conflated.
var errUnknownClientKey = errors.New("fake key store: unknown client key")

// fakeKeyStore is an in-memory KeyStore for tests only.
type fakeKeyStore struct {
	pubs    map[string]crypto.PublicKey
	secrets map[string]string
}

func (f fakeKeyStore) PublicKey(clientKey string) (crypto.PublicKey, error) {
	pub, ok := f.pubs[clientKey]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errUnknownClientKey, clientKey)
	}
	return pub, nil
}

func (f fakeKeyStore) ClientSecret(clientKey string) (string, error) {
	secret, ok := f.secrets[clientKey]
	if !ok {
		return "", fmt.Errorf("%w: %s", errUnknownClientKey, clientKey)
	}
	return secret, nil
}

const (
	verifyTestClientA = "client-a"
	verifyTestClientB = "client-b"
	verifyTestSecretA = "secret-a"
	verifyTestTime    = "2026-09-10T10:00:00.000+07:00"
)

func newVerifyTestStore() fakeKeyStore {
	keys := testRSAKeys()
	return fakeKeyStore{
		pubs: map[string]crypto.PublicKey{
			verifyTestClientA: keys[0].Public(),
			verifyTestClientB: keys[1].Public(),
		},
		secrets: map[string]string{
			verifyTestClientA: verifyTestSecretA,
			// client-b intentionally has a different secret, so signing with
			// A's secret and pointing ClientKey at B fails verification
			// rather than accidentally succeeding.
			verifyTestClientB: "secret-b",
		},
	}
}

func validAccessTokenRequest() IncomingRequest {
	keys := testRSAKeys()
	stringToSign := BuildStringToSignAccessToken(verifyTestClientA, verifyTestTime)
	sig, err := SignAsymmetric(keys[0], stringToSign)
	if err != nil {
		panic(err)
	}
	return IncomingRequest{
		Timestamp: verifyTestTime,
		ClientKey: verifyTestClientA,
		Signature: sig,
	}
}

func validTransactionRequest(t *testing.T, symmetric bool) IncomingRequest {
	t.Helper()
	req := IncomingRequest{
		Method:      "POST",
		EndpointURL: "https://openapi.example.com/v1.0/transfer-va",
		Body:        []byte(`{"amount":"10000.00"}`),
		Timestamp:   verifyTestTime,
		ClientKey:   verifyTestClientA,
		AccessToken: "opaque-access-token",
	}
	stringToSign := BuildStringToSignTransaction(req.Method, req.EndpointURL, req.AccessToken, req.Body, req.Timestamp, symmetric)
	if symmetric {
		req.Signature = SignSymmetric(verifyTestSecretA, stringToSign)
		return req
	}
	keys := testRSAKeys()
	sig, err := SignAsymmetric(keys[0], stringToSign)
	if err != nil {
		t.Fatalf("SignAsymmetric: %v", err)
	}
	req.Signature = sig
	return req
}

func TestServerVerifier_VerifyAccessTokenRequest_RoundTrip(t *testing.T) {
	store := newVerifyTestStore()
	// Freshness is exercised separately (TestServerVerifier_TimestampFreshness);
	// disabled here so this test's fixed fixture timestamp doesn't depend on
	// when it happens to run.
	v := &ServerVerifier{KeyStore: store, TimestampWindow: DisableTimestampFreshnessCheck}

	tests := []struct {
		name         string
		mutate       func(IncomingRequest) IncomingRequest
		wantErr      bool
		wantMismatch bool // asserted via errors.Is(err, ErrSignatureMismatch) when true
	}{
		{"valid", func(r IncomingRequest) IncomingRequest { return r }, false, false},
		{"tampered signature", func(r IncomingRequest) IncomingRequest {
			r.Signature = tamperHex(r.Signature)
			return r
		}, true, true},
		{"tampered timestamp", func(r IncomingRequest) IncomingRequest {
			r.Timestamp = "2026-09-10T10:00:01.000+07:00"
			return r
		}, true, true},
		{"client key points at different key", func(r IncomingRequest) IncomingRequest {
			r.ClientKey = verifyTestClientB
			return r
		}, true, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.mutate(validAccessTokenRequest())
			err := v.VerifyAccessTokenRequest(req)
			if tc.wantErr && err == nil {
				t.Fatal("want error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("want no error, got %v", err)
			}
			if tc.wantMismatch && !errors.Is(err, ErrSignatureMismatch) {
				t.Errorf("want errors.Is(err, ErrSignatureMismatch), got %v", err)
			}
		})
	}
}

func TestServerVerifier_VerifyTransactionRequest_RoundTrip(t *testing.T) {
	store := newVerifyTestStore()

	for _, symmetric := range []bool{true, false} {
		mode := SignatureModeAsymmetric
		if symmetric {
			mode = SignatureModeSymmetric
		}
		t.Run(fmt.Sprintf("symmetric=%v", symmetric), func(t *testing.T) {
			v := &ServerVerifier{KeyStore: store, Mode: mode, TimestampWindow: DisableTimestampFreshnessCheck}

			tests := []struct {
				name         string
				mutate       func(IncomingRequest) IncomingRequest
				wantErr      bool
				wantMismatch bool
			}{
				{"valid", func(r IncomingRequest) IncomingRequest { return r }, false, false},
				{"tampered signature", func(r IncomingRequest) IncomingRequest {
					r.Signature = tamperHex(r.Signature)
					return r
				}, true, true},
				{"tampered body", func(r IncomingRequest) IncomingRequest {
					r.Body = []byte(`{"amount":"99999.00"}`)
					return r
				}, true, true},
				{"tampered timestamp", func(r IncomingRequest) IncomingRequest {
					r.Timestamp = "2026-09-10T10:00:01.000+07:00"
					return r
				}, true, true},
				{"client key points at different key/secret", func(r IncomingRequest) IncomingRequest {
					r.ClientKey = verifyTestClientB
					return r
				}, true, true},
			}

			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					req := tc.mutate(validTransactionRequest(t, symmetric))
					err := v.VerifyTransactionRequest(req)
					if tc.wantErr && err == nil {
						t.Fatal("want error, got nil")
					}
					if !tc.wantErr && err != nil {
						t.Fatalf("want no error, got %v", err)
					}
					if tc.wantMismatch && !errors.Is(err, ErrSignatureMismatch) {
						t.Errorf("want errors.Is(err, ErrSignatureMismatch), got %v", err)
					}
				})
			}
		})
	}
}

func TestServerVerifier_VerifyTransactionRequest_AsymmetricIgnoresAccessToken(t *testing.T) {
	// The asymmetric formula excludes AccessToken entirely, so a garbage
	// AccessToken on an otherwise-correctly-signed asymmetric request must
	// still verify.
	store := newVerifyTestStore()
	v := &ServerVerifier{KeyStore: store, Mode: SignatureModeAsymmetric, TimestampWindow: DisableTimestampFreshnessCheck}
	req := validTransactionRequest(t, false)
	req.AccessToken = "this-is-not-part-of-the-asymmetric-formula"
	if err := v.VerifyTransactionRequest(req); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
}

func TestServerVerifier_KeyStoreLookupFailureIsNotSwallowed(t *testing.T) {
	store := newVerifyTestStore()

	t.Run("access token unknown client key", func(t *testing.T) {
		v := &ServerVerifier{KeyStore: store, TimestampWindow: DisableTimestampFreshnessCheck}
		req := validAccessTokenRequest()
		req.ClientKey = "unknown-client"
		err := v.VerifyAccessTokenRequest(req)
		if err == nil {
			t.Fatal("want error, got nil")
		}
		if !errors.Is(err, errUnknownClientKey) {
			t.Fatalf("want errors.Is(err, errUnknownClientKey), got %v", err)
		}
		if errors.Is(err, ErrSignatureMismatch) {
			t.Fatalf("lookup failure must not be conflated with ErrSignatureMismatch, got %v", err)
		}
	})

	t.Run("transaction symmetric unknown client key", func(t *testing.T) {
		v := &ServerVerifier{KeyStore: store, Mode: SignatureModeSymmetric, TimestampWindow: DisableTimestampFreshnessCheck}
		req := validTransactionRequest(t, true)
		req.ClientKey = "unknown-client"
		err := v.VerifyTransactionRequest(req)
		if err == nil {
			t.Fatal("want error, got nil")
		}
		if !errors.Is(err, errUnknownClientKey) {
			t.Fatalf("want errors.Is(err, errUnknownClientKey), got %v", err)
		}
	})

	t.Run("transaction asymmetric unknown client key", func(t *testing.T) {
		v := &ServerVerifier{KeyStore: store, Mode: SignatureModeAsymmetric, TimestampWindow: DisableTimestampFreshnessCheck}
		req := validTransactionRequest(t, false)
		req.ClientKey = "unknown-client"
		err := v.VerifyTransactionRequest(req)
		if err == nil {
			t.Fatal("want error, got nil")
		}
		if !errors.Is(err, errUnknownClientKey) {
			t.Fatalf("want errors.Is(err, errUnknownClientKey), got %v", err)
		}
	})
}

// TestServerVerifier_EmptyClientSecretIsNotConflatedWithMismatch pins that
// a KeyStore returning "" for ClientSecret (a config mistake, not evidence
// of a tampered request) surfaces as ErrEmptyClientSecret, distinguishable
// from ErrSignatureMismatch.
func TestServerVerifier_EmptyClientSecretIsNotConflatedWithMismatch(t *testing.T) {
	store := fakeKeyStore{secrets: map[string]string{verifyTestClientA: ""}}
	v := &ServerVerifier{KeyStore: store, Mode: SignatureModeSymmetric, TimestampWindow: DisableTimestampFreshnessCheck}
	req := validTransactionRequest(t, true)

	err := v.VerifyTransactionRequest(req)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if !errors.Is(err, ErrEmptyClientSecret) {
		t.Fatalf("want errors.Is(err, ErrEmptyClientSecret), got %v", err)
	}
	if errors.Is(err, ErrSignatureMismatch) {
		t.Fatalf("empty-secret config error must not be conflated with ErrSignatureMismatch, got %v", err)
	}
}

// TestServerVerifier_UnusableRSAKeyIsNotConflatedWithMismatch pins that a
// KeyStore returning a non-RSA public key (a config mistake) surfaces as
// ErrNotRSASigner, distinguishable from ErrSignatureMismatch, for both
// VerifyTransactionRequest and VerifyAccessTokenRequest.
func TestServerVerifier_UnusableRSAKeyIsNotConflatedWithMismatch(t *testing.T) {
	store := fakeKeyStore{pubs: map[string]crypto.PublicKey{verifyTestClientA: "not-an-rsa-key"}}

	t.Run("transaction", func(t *testing.T) {
		v := &ServerVerifier{KeyStore: store, Mode: SignatureModeAsymmetric, TimestampWindow: DisableTimestampFreshnessCheck}
		req := validTransactionRequest(t, false)
		err := v.VerifyTransactionRequest(req)
		if !errors.Is(err, ErrNotRSASigner) {
			t.Fatalf("want errors.Is(err, ErrNotRSASigner), got %v", err)
		}
		if errors.Is(err, ErrSignatureMismatch) {
			t.Fatalf("unusable-key config error must not be conflated with ErrSignatureMismatch, got %v", err)
		}
	})

	t.Run("access token", func(t *testing.T) {
		v := &ServerVerifier{KeyStore: store, TimestampWindow: DisableTimestampFreshnessCheck}
		req := validAccessTokenRequest()
		err := v.VerifyAccessTokenRequest(req)
		if !errors.Is(err, ErrNotRSASigner) {
			t.Fatalf("want errors.Is(err, ErrNotRSASigner), got %v", err)
		}
		if errors.Is(err, ErrSignatureMismatch) {
			t.Fatalf("unusable-key config error must not be conflated with ErrSignatureMismatch, got %v", err)
		}
	})
}

// TestServerVerifier_NilKeyStoreDoesNotPanic is the regression test for a
// GAN evaluator finding: a zero-value ServerVerifier{} (no KeyStore set)
// used to panic with a nil pointer dereference on the first lookup instead
// of returning a clean error, inconsistent with the nil-guards already
// present for Profile/Now on the same struct.
func TestServerVerifier_NilKeyStoreDoesNotPanic(t *testing.T) {
	v := &ServerVerifier{}

	if err := v.VerifyAccessTokenRequest(validAccessTokenRequest()); !errors.Is(err, ErrNoKeyStore) {
		t.Errorf("VerifyAccessTokenRequest() with nil KeyStore: err = %v, want errors.Is(err, ErrNoKeyStore)", err)
	}
	if err := v.VerifyTransactionRequest(validTransactionRequest(t, true)); !errors.Is(err, ErrNoKeyStore) {
		t.Errorf("VerifyTransactionRequest() with nil KeyStore: err = %v, want errors.Is(err, ErrNoKeyStore)", err)
	}
}

func TestServerVerifier_TimestampFreshness(t *testing.T) {
	store := newVerifyTestStore()
	fixedNow, err := time.Parse(DefaultTimestampLayout, verifyTestTime)
	if err != nil {
		t.Fatalf("time.Parse: %v", err)
	}

	accessTokenAt := func(offset time.Duration) IncomingRequest {
		ts := fixedNow.Add(offset).Format(DefaultTimestampLayout)
		stringToSign := BuildStringToSignAccessToken(verifyTestClientA, ts)
		keys := testRSAKeys()
		sig, err := SignAsymmetric(keys[0], stringToSign)
		if err != nil {
			t.Fatalf("SignAsymmetric: %v", err)
		}
		return IncomingRequest{Timestamp: ts, ClientKey: verifyTestClientA, Signature: sig}
	}
	transactionAt := func(offset time.Duration) IncomingRequest {
		req := validTransactionRequest(t, true)
		ts := fixedNow.Add(offset).Format(DefaultTimestampLayout)
		req.Timestamp = ts
		stringToSign := BuildStringToSignTransaction(req.Method, req.EndpointURL, req.AccessToken, req.Body, ts, true)
		req.Signature = SignSymmetric(verifyTestSecretA, stringToSign)
		return req
	}

	tests := []struct {
		name    string
		window  time.Duration
		offset  time.Duration
		wantErr bool
	}{
		{"within window, past", 5 * time.Minute, -1 * time.Minute, false},
		{"within window, future", 5 * time.Minute, 1 * time.Minute, false},
		{"outside window, past", 5 * time.Minute, -10 * time.Minute, true},
		{"outside window, future", 5 * time.Minute, 10 * time.Minute, true},
		{"disable sentinel accepts wildly old timestamp", DisableTimestampFreshnessCheck, -24 * time.Hour, false},
		{"zero value uses DefaultTimestampWindow: within default, accepted", 0, -1 * time.Minute, false},
		{"zero value uses DefaultTimestampWindow: outside default, rejected", 0, -10 * time.Minute, true},
	}

	methods := []struct {
		name string
		sign func(time.Duration) IncomingRequest
		call func(*ServerVerifier, IncomingRequest) error
	}{
		{"VerifyAccessTokenRequest", accessTokenAt, (*ServerVerifier).VerifyAccessTokenRequest},
		{"VerifyTransactionRequest", transactionAt, (*ServerVerifier).VerifyTransactionRequest},
	}

	for _, m := range methods {
		t.Run(m.name, func(t *testing.T) {
			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					v := &ServerVerifier{
						KeyStore:        store,
						Mode:            SignatureModeSymmetric,
						TimestampWindow: tc.window,
						Now:             func() time.Time { return fixedNow },
					}
					err := m.call(v, m.sign(tc.offset))
					if tc.wantErr && err == nil {
						t.Fatal("want error, got nil")
					}
					if !tc.wantErr && err != nil {
						t.Fatalf("want no error, got %v", err)
					}
				})
			}
		})
	}
}

func TestServerVerifier_ProfileHookAffectsTimestampParsing(t *testing.T) {
	// fixedLayoutProfile (defined in header_test.go) overrides TimestampLayout
	// to "2006" — proves checkFreshness actually consults v.profile() rather
	// than hardcoding DefaultTimestampLayout.
	store := newVerifyTestStore()
	fixedNow := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	profile := fixedLayoutProfile{}
	timestamp := fixedNow.Format(profile.TimestampLayout()) // "2026"

	stringToSign := BuildStringToSignAccessToken(verifyTestClientA, timestamp)
	keys := testRSAKeys()
	sig, err := SignAsymmetric(keys[0], stringToSign)
	if err != nil {
		t.Fatalf("SignAsymmetric: %v", err)
	}
	req := IncomingRequest{Timestamp: timestamp, ClientKey: verifyTestClientA, Signature: sig}

	v := &ServerVerifier{
		KeyStore:        store,
		Profile:         profile,
		TimestampWindow: time.Hour,
		Now:             func() time.Time { return fixedNow },
	}
	if err := v.VerifyAccessTokenRequest(req); err != nil {
		t.Fatalf("want no error with matching custom Profile layout, got %v", err)
	}

	// Without the custom Profile, "2026" fails to parse under
	// DefaultTimestampLayout, proving the hook is load-bearing rather than a
	// no-op.
	v.Profile = nil
	if err := v.VerifyAccessTokenRequest(req); err == nil {
		t.Fatal("want error when timestamp doesn't match DefaultTimestampLayout, got nil")
	}
}

func TestServerVerifier_VerifyAccessTokenRequest_IgnoresMode(t *testing.T) {
	store := newVerifyTestStore()
	req := validAccessTokenRequest()

	for _, mode := range []SignatureMode{SignatureModeSymmetric, SignatureModeAsymmetric} {
		t.Run(fmt.Sprintf("mode=%v", mode), func(t *testing.T) {
			v := &ServerVerifier{KeyStore: store, Mode: mode, TimestampWindow: DisableTimestampFreshnessCheck}
			if err := v.VerifyAccessTokenRequest(req); err != nil {
				t.Fatalf("want no error regardless of Mode, got %v", err)
			}
		})
	}
}
