package snap

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"strings"
	"sync"
	"testing"
)

// sha256HexOfEmptyBody is printf ” | sha256sum, hardcoded independently of
// the implementation under test so a broken implementation can't
// tautologically agree with itself.
const sha256HexOfEmptyBody = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

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
		{"tampered signature fails", secret, stringToSign, sig[:len(sig)-1] + "0", false},
		{"tampered stringToSign fails", secret, stringToSign + "x", sig, false},
		{"wrong secret fails", "other-secret", stringToSign, sig, false},
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

func TestSignAsymmetricProducesLowercaseHex(t *testing.T) {
	key := testRSAKeys()[0]
	sig, err := SignAsymmetric(key, "some string to sign")
	if err != nil {
		t.Fatalf("SignAsymmetric() error = %v", err)
	}
	if sig != strings.ToLower(sig) {
		t.Errorf("SignAsymmetric() output not lowercase: %q", sig)
	}
	if _, err := hex.DecodeString(sig); err != nil {
		t.Errorf("SignAsymmetric() output not valid hex: %v", err)
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
		{"tampered signature fails", &key.PublicKey, stringToSign, sig[:len(sig)-2] + "00", true},
		{"tampered stringToSign fails", &key.PublicKey, stringToSign + "x", sig, true},
		{"wrong key fails", &otherKey.PublicKey, stringToSign, sig, true},
		{"non-hex signature fails", &key.PublicKey, stringToSign, "not-hex!!", true},
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
	if err := VerifyAsymmetric(pub, "x", "00"); err == nil {
		t.Error("VerifyAsymmetric() with non-RSA public key: want error, got nil")
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
