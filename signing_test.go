package snap

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
)

// sha256HexOfEmptyBody is printf ” | sha256sum, hardcoded independently of
// the implementation under test so a broken implementation can't
// tautologically agree with itself.
const sha256HexOfEmptyBody = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

// tamperSig flips one bit of a Base64 signature's decoded bytes and
// re-encodes it, so the result still decodes (the tamper is caught by the
// cryptographic check, not by a decoding error) and never equals the original.
func tamperSig(sig string) string {
	b, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		panic(err)
	}
	b[0] ^= 0x01
	return base64.StdEncoding.EncodeToString(b)
}

// legacyHex re-encodes a Base64 signature as lowercase hex, the v0.1.x format.
func legacyHex(sig string) string {
	b, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

var testRSAKeys = sync.OnceValue(func() [2]*rsa.PrivateKey {
	k1, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	k2, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	return [2]*rsa.PrivateKey{k1, k2}
})

func TestSignVerifySymmetric(t *testing.T) {
	const secret = "s3cr3t"
	const stringToSign = "POST:/v1.0/access-token/b2b:2026-09-10T10:00:00.000+07:00"

	sig := SignSymmetric(secret, stringToSign)

	tests := []struct {
		name         string
		secret       string
		stringToSign string
		signature    string
		want         bool
	}{
		{"correct signature verifies", secret, stringToSign, sig, true},
		{"tampered signature fails", secret, stringToSign, tamperSig(sig), false},
		{"tampered stringToSign fails", secret, stringToSign + "x", sig, false},
		{"wrong secret fails", "other-secret", stringToSign, sig, false},
		{"v0.1.x hex signature still verifies", secret, stringToSign, legacyHex(sig), true},
		{"uppercase-hex signature still verifies", secret, stringToSign, strings.ToUpper(legacyHex(sig)), true},
		{"truncated signature fails", secret, stringToSign, sig[:40], false},
		{"undecodable signature fails closed, not panics", secret, stringToSign, "not-a-signature!!", false},
		{
			"empty clientSecret rejected, not treated as a valid HMAC key",
			"", stringToSign, SignSymmetric("", stringToSign), false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VerifySymmetric(tt.secret, tt.stringToSign, tt.signature)
			if got != tt.want {
				t.Errorf("VerifySymmetric() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSignSymmetricKnownAnswer checks SignSymmetric against a signature
// value computed independently (Python stdlib hmac+hashlib), not by
// round-tripping through this package's own functions — a round-trip-only
// test can't detect a paired algorithm substitution (e.g. accidentally
// switching to SHA-384 or hex output) since it would still agree with
// itself.
func TestSignSymmetricKnownAnswer(t *testing.T) {
	const secret = "s3cr3t"
	const stringToSign = "POST:/v1.0/access-token/b2b:2026-09-10T10:00:00.000+07:00"
	const want = "QAklk7fG5uW/SgnvNG2DEIn+O7Ru/N/xqRVF4PAesGTFsnRbk7FXOJw5xkqENsLovAew2d5yP7dsOcS4xsQ2/w=="

	got := SignSymmetric(secret, stringToSign)
	if got != want {
		t.Errorf("SignSymmetric() = %q, want independently-computed %q", got, want)
	}
}

func TestSignAsymmetricProducesBase64(t *testing.T) {
	key := testRSAKeys()[0]
	sig, err := SignAsymmetric(key, "some string to sign")
	if err != nil {
		t.Fatalf("SignAsymmetric() error = %v", err)
	}
	b, err := base64.StdEncoding.DecodeString(sig)
	if err != nil || len(b) != key.Size() {
		t.Errorf("SignAsymmetric() output is not standard Base64 of %d bytes: %q %v", key.Size(), sig, err)
	}
}

func TestSignVerifyAsymmetric(t *testing.T) {
	keys := testRSAKeys()
	key, otherKey := keys[0], keys[1]
	const stringToSign = "POST:/v1.0/transfer-va:token123:" + sha256HexOfEmptyBody + ":2026-09-10T10:00:00.000+07:00"

	sig, err := SignAsymmetric(key, stringToSign)
	if err != nil {
		t.Fatalf("SignAsymmetric() error = %v", err)
	}

	tests := []struct {
		name         string
		pub          any
		stringToSign string
		signature    string
		wantErr      bool
	}{
		{"correct signature verifies", &key.PublicKey, stringToSign, sig, false},
		{"tampered signature fails", &key.PublicKey, stringToSign, tamperSig(sig), true},
		{"v0.1.x hex signature still verifies", &key.PublicKey, stringToSign, legacyHex(sig), false},
		{"tampered stringToSign fails", &key.PublicKey, stringToSign + "x", sig, true},
		{"wrong key fails", &otherKey.PublicKey, stringToSign, sig, true},
		{"undecodable signature fails", &key.PublicKey, stringToSign, "not-a-signature!!", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyAsymmetric(tt.pub, tt.stringToSign, tt.signature)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyAsymmetric() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVerifyAsymmetricRejectsNonRSAKey(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey() error = %v", err)
	}
	if err := VerifyAsymmetric(pub, "x", "00"); !errors.Is(err, ErrNotRSASigner) {
		t.Errorf("VerifyAsymmetric() with non-RSA public key: err = %v, want errors.Is(err, ErrNotRSASigner)", err)
	}
}

func TestSignAsymmetricRejectsNonRSASigner(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey() error = %v", err)
	}
	if _, err := SignAsymmetric(priv, "x"); !errors.Is(err, ErrNotRSASigner) {
		t.Errorf("SignAsymmetric() with non-RSA signer: err = %v, want errors.Is(err, ErrNotRSASigner)", err)
	}
}

func TestSignVerifyAsymmetricRejectWeakKey(t *testing.T) {
	weakKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("rsa.GenerateKey(1024) error = %v", err)
	}
	if _, err := SignAsymmetric(weakKey, "x"); !errors.Is(err, ErrWeakRSAKey) {
		t.Errorf("SignAsymmetric() with 1024-bit key: err = %v, want errors.Is(err, ErrWeakRSAKey)", err)
	}

	// A signature produced by a strong key must still be rejected on the
	// verify side if the caller is (mis)configured to check against a weak
	// public key — the floor applies independently on each side.
	strongKey := testRSAKeys()[0]
	sig, err := SignAsymmetric(strongKey, "x")
	if err != nil {
		t.Fatalf("SignAsymmetric() error = %v", err)
	}
	if err := VerifyAsymmetric(&weakKey.PublicKey, "x", sig); !errors.Is(err, ErrWeakRSAKey) {
		t.Errorf("VerifyAsymmetric() with 1024-bit public key: err = %v, want errors.Is(err, ErrWeakRSAKey)", err)
	}
}

// TestSignVerifyAsymmetricRejectsZeroValuePublicKey is the regression test
// for a security review finding: a *rsa.PublicKey with a nil N field (e.g.
// &rsa.PublicKey{} returned by a buggy KeyStore before it's populated)
// panicked on rsaPub.N.BitLen() instead of failing closed with an error.
func TestSignVerifyAsymmetricRejectsZeroValuePublicKey(t *testing.T) {
	zeroPub := &rsa.PublicKey{}
	if err := VerifyAsymmetric(zeroPub, "x", "00"); !errors.Is(err, ErrNotRSASigner) {
		t.Errorf("VerifyAsymmetric() with zero-value public key: err = %v, want errors.Is(err, ErrNotRSASigner)", err)
	}

	zeroSigner := zeroValueSigner{}
	if _, err := SignAsymmetric(zeroSigner, "x"); !errors.Is(err, ErrNotRSASigner) {
		t.Errorf("SignAsymmetric() with a signer whose Public() returns a zero-value key: err = %v, want errors.Is(err, ErrNotRSASigner)", err)
	}
}

// zeroValueSigner is a crypto.Signer whose Public() returns a *rsa.PublicKey
// with a nil N — simulating a buggy caller-supplied Signer implementation,
// for TestSignVerifyAsymmetricRejectsZeroValuePublicKey.
type zeroValueSigner struct{}

func (zeroValueSigner) Public() crypto.PublicKey { return &rsa.PublicKey{} }
func (zeroValueSigner) Sign(io.Reader, []byte, crypto.SignerOpts) ([]byte, error) {
	panic("not reached: rejected before Sign is called")
}

// TestSignAsymmetricIsDeterministic asserts SHA256withRSA (PKCS#1 v1.5)
// signing produces identical output across repeated calls for the same
// input. This is true for PKCS#1 v1.5 and false for RSA-PSS (which is
// randomized) — a determinism regression here would mean the implementation
// silently drifted to PSS, which a pure round-trip test cannot detect.
func TestSignAsymmetricIsDeterministic(t *testing.T) {
	key := testRSAKeys()[0]
	sig1, err := SignAsymmetric(key, "deterministic check")
	if err != nil {
		t.Fatalf("SignAsymmetric() error = %v", err)
	}
	sig2, err := SignAsymmetric(key, "deterministic check")
	if err != nil {
		t.Fatalf("SignAsymmetric() error = %v", err)
	}
	if sig1 != sig2 {
		t.Errorf("SignAsymmetric() not deterministic: %q != %q (expected for PKCS#1 v1.5; would legitimately differ under RSA-PSS)", sig1, sig2)
	}
}

func TestBuildStringToSignAccessToken(t *testing.T) {
	got := BuildStringToSignAccessToken("client-123", "2026-09-10T10:00:00.000+07:00")
	want := "client-123|2026-09-10T10:00:00.000+07:00"
	if got != want {
		t.Errorf("BuildStringToSignAccessToken() = %q, want %q", got, want)
	}
}

func TestBuildStringToSignTransaction(t *testing.T) {
	const method = "POST"
	const endpointURL = "/v1.0/transfer-va/create-va"
	const accessToken = "abc.def.ghi"
	const timestamp = "2026-09-10T10:00:00.000+07:00"
	body := []byte(`{"amount":"10000.00"}`)

	symmetricStr := BuildStringToSignTransaction(method, endpointURL, accessToken, body, timestamp, true)
	asymmetricStr := BuildStringToSignTransaction(method, endpointURL, accessToken, body, timestamp, false)

	// bodyHashHex is computed independently of the implementation under
	// test via the stdlib sha256/hex packages directly, so a broken
	// implementation can't tautologically agree with a wrong expectation.
	bodyHash := sha256.Sum256(body)
	bodyHashHex := hex.EncodeToString(bodyHash[:])

	wantSymmetric := method + ":" + endpointURL + ":" + accessToken + ":" + bodyHashHex + ":" + timestamp
	wantAsymmetric := method + ":" + endpointURL + ":" + bodyHashHex + ":" + timestamp

	if symmetricStr != wantSymmetric {
		t.Errorf("symmetric stringToSign = %q, want %q", symmetricStr, wantSymmetric)
	}
	if asymmetricStr != wantAsymmetric {
		t.Errorf("asymmetric stringToSign = %q, want %q", asymmetricStr, wantAsymmetric)
	}

	// The two forms must differ by exactly the ":"+AccessToken segment:
	// removing it from the symmetric form reproduces the asymmetric form.
	reconstructed := strings.Replace(symmetricStr, ":"+accessToken+":", ":", 1)
	if reconstructed != asymmetricStr {
		t.Errorf("symmetric form minus AccessToken segment = %q, want asymmetric form %q", reconstructed, asymmetricStr)
	}
	if strings.Contains(asymmetricStr, accessToken) {
		t.Errorf("asymmetric stringToSign must not contain AccessToken, got %q", asymmetricStr)
	}
}

// TestBuildStringToSignTransactionMinifiesBody: the standard hashes
// minify(RequestBody), so a pretty-printed body signs like its compact form
// (whitespace inside strings is kept). The hash is computed independently
// (Python hashlib over the compact bytes).
func TestBuildStringToSignTransactionMinifiesBody(t *testing.T) {
	const compactHash = "31505dec19d6ac9caadcfd51fb5831dc172cc6f924b70f28442e09f97537fac4"
	pretty := []byte("{\n  \"amount\": \"10000.00\",\n\t\"note\" : \"a b\"\n}\n")
	compact := []byte(`{"amount":"10000.00","note":"a b"}`)
	want := "POST:/v1.0/transfer-interbank:token:" + compactHash + ":2026-09-10T10:00:00.000+07:00"
	for name, body := range map[string][]byte{"pretty": pretty, "compact": compact} {
		if got := BuildStringToSignTransaction("POST", "/v1.0/transfer-interbank", "token", body, "2026-09-10T10:00:00.000+07:00", true); got != want {
			t.Errorf("%s body: %q, want %q", name, got, want)
		}
	}
	notJSON := []byte("a=1 b=2")
	h := sha256.Sum256(notJSON)
	if got := BuildStringToSignTransaction("POST", "/x", "", notJSON, "ts", false); got != "POST:/x:"+hex.EncodeToString(h[:])+":ts" {
		t.Errorf("a non-JSON body is hashed as is: %q", got)
	}
}

func TestBuildStringToSignTransactionEmptyBody(t *testing.T) {
	got := BuildStringToSignTransaction("GET", "/v1.0/balance-inquiry", "token", nil, "2026-09-10T10:00:00.000+07:00", true)
	want := "GET:/v1.0/balance-inquiry:token:" + sha256HexOfEmptyBody + ":2026-09-10T10:00:00.000+07:00"
	if got != want {
		t.Errorf("BuildStringToSignTransaction() with empty body = %q, want %q", got, want)
	}

	// []byte{} must hash identically to nil.
	got2 := BuildStringToSignTransaction("GET", "/v1.0/balance-inquiry", "token", []byte{}, "2026-09-10T10:00:00.000+07:00", true)
	if got2 != want {
		t.Errorf("BuildStringToSignTransaction() with []byte{} body = %q, want %q", got2, want)
	}
}

func TestParseRSAPrivateKeyPEM(t *testing.T) {
	rsaKey := testRSAKeys()[0]

	pkcs1PEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(rsaKey),
	})

	pkcs8Bytes, err := x509.MarshalPKCS8PrivateKey(rsaKey)
	if err != nil {
		t.Fatalf("MarshalPKCS8PrivateKey() error = %v", err)
	}
	pkcs8PEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: pkcs8Bytes,
	})

	_, edPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey() error = %v", err)
	}
	edBytes, err := x509.MarshalPKCS8PrivateKey(edPriv)
	if err != nil {
		t.Fatalf("MarshalPKCS8PrivateKey() error = %v", err)
	}
	edPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: edBytes,
	})

	tests := []struct {
		name    string
		pemData []byte
		wantErr error // nil means "no error expected"
	}{
		{"valid PKCS#1 RSA key", pkcs1PEM, nil},
		{"valid PKCS#8 RSA key", pkcs8PEM, nil},
		{"malformed PEM", []byte("this is not a pem block"), ErrNoPEMBlock},
		{"non-RSA key type", edPEM, ErrNotRSAKey},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer, err := ParseRSAPrivateKeyPEM(tt.pemData)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ParseRSAPrivateKeyPEM() error = %v, want errors.Is(err, %v)", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseRSAPrivateKeyPEM() error = %v, want nil", err)
			}
			if signer == nil {
				t.Fatal("ParseRSAPrivateKeyPEM() returned nil signer with nil error")
			}
			// Prove it's genuinely usable as a crypto.Signer end to end.
			sig, err := SignAsymmetric(signer, "roundtrip check")
			if err != nil {
				t.Fatalf("SignAsymmetric() with parsed key error = %v", err)
			}
			if err := VerifyAsymmetric(signer.Public(), "roundtrip check", sig); err != nil {
				t.Errorf("VerifyAsymmetric() with parsed key's public part error = %v", err)
			}
		})
	}
}
