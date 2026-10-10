package snap

import (
	"bytes"
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
)

// DefaultTimestampLayout is the Go time layout equivalent to the SNAP
// standard's yyyy-MM-ddTHH:mm:ss.SSSTZD timestamp format.
const DefaultTimestampLayout = "2006-01-02T15:04:05.000-07:00"

// Sentinel errors returned by ParseRSAPrivateKeyPEM, wrapped with %w so
// callers can distinguish failure modes via errors.Is.
var (
	ErrNoPEMBlock = errors.New("snap: no PEM block found")
	ErrNotRSAKey  = errors.New("snap: not an RSA key")
)

// Sentinel errors returned by SignAsymmetric and VerifyAsymmetric, wrapped
// with %w so callers can distinguish failure modes via errors.Is.
var (
	ErrNotRSASigner = errors.New("snap: not an RSA signer")
	ErrWeakRSAKey   = errors.New("snap: RSA key smaller than minimum size")
	// ErrSignatureEncoding: the signature is neither Base64 nor hex of the expected size.
	ErrSignatureEncoding = errors.New("snap: signature is not Base64 (or hex) of the expected size")
)

// minRSAKeyBits is the minimum RSA modulus size accepted for SHA256withRSA
// signing and verification. The standard specifies 256-bit (i.e. 2048-bit
// modulus) keys; this floor rejects weaker keys that would be practically
// forgeable, most importantly on the verify side where the public key may
// come from a partner-registered certificate.
const minRSAKeyBits = 2048

// SignSymmetric computes the HMAC-SHA512 signature of stringToSign using
// clientSecret as the key, returning standard Base64 (the encoding SNAP
// participants use for X-SIGNATURE).
func SignSymmetric(clientSecret, stringToSign string) string {
	return base64.StdEncoding.EncodeToString(hmacSHA512(clientSecret, stringToSign))
}

func hmacSHA512(clientSecret, stringToSign string) []byte {
	mac := hmac.New(sha512.New, []byte(clientSecret))
	mac.Write([]byte(stringToSign))
	return mac.Sum(nil)
}

// decodeSignature decodes an X-SIGNATURE of size bytes: standard Base64, or
// hex as produced by v0.1.x of this package. The two cannot be confused
// because their lengths for the same size differ.
func decodeSignature(signature string, size int) ([]byte, error) {
	if b, err := base64.StdEncoding.DecodeString(signature); err == nil && len(b) == size {
		return b, nil
	}
	if b, err := hex.DecodeString(signature); err == nil && len(b) == size {
		return b, nil
	}
	return nil, ErrSignatureEncoding
}

// VerifySymmetric reports whether signature is the valid HMAC-SHA512
// signature of stringToSign under clientSecret, using a constant-time
// comparison.
func VerifySymmetric(clientSecret, stringToSign, signature string) bool {
	if clientSecret == "" {
		// An empty key turns HMAC into a deterministic, publicly computable
		// MAC — a KeyStore implementation mistake that returns "" for an
		// unregistered client must not be treated as "verifies against the
		// empty string," so this is rejected before ever reaching hmac.Equal.
		return false
	}
	got, err := decodeSignature(signature, sha512.Size)
	if err != nil {
		return false
	}
	return hmac.Equal(hmacSHA512(clientSecret, stringToSign), got)
}

// SignAsymmetric signs stringToSign with SHA256withRSA (PKCS#1 v1.5) using
// signer, returning standard Base64. signer is a crypto.Signer
// rather than a concrete *rsa.PrivateKey so that HSM/KMS-backed keys can be
// plugged in without an API change.
func SignAsymmetric(signer crypto.Signer, stringToSign string) (string, error) {
	if signer == nil {
		return "", fmt.Errorf("snap: sign asymmetric: %w", ErrNotRSASigner)
	}
	rsaPub, ok := signer.Public().(*rsa.PublicKey)
	if !ok || rsaPub == nil || rsaPub.N == nil {
		return "", fmt.Errorf("snap: sign asymmetric: %w", ErrNotRSASigner)
	}
	if rsaPub.N.BitLen() < minRSAKeyBits {
		return "", fmt.Errorf("snap: sign asymmetric: %w", ErrWeakRSAKey)
	}
	digest := sha256.Sum256([]byte(stringToSign))
	sig, err := signer.Sign(rand.Reader, digest[:], crypto.SHA256)
	if err != nil {
		return "", fmt.Errorf("snap: sign asymmetric: %w", err)
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

// VerifyAsymmetric verifies a SHA256withRSA signature of stringToSign
// (standard Base64, or hex from v0.1.x) against pub, which must be an
// *rsa.PublicKey. It returns a non-nil error if pub is not an RSA key, the
// signature is not decodable, or it does not verify.
func VerifyAsymmetric(pub crypto.PublicKey, stringToSign, signature string) error {
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok || rsaPub == nil || rsaPub.N == nil {
		return fmt.Errorf("snap: verify asymmetric: %w", ErrNotRSASigner)
	}
	if rsaPub.N.BitLen() < minRSAKeyBits {
		return fmt.Errorf("snap: verify asymmetric: %w", ErrWeakRSAKey)
	}
	sig, err := decodeSignature(signature, rsaPub.Size())
	if err != nil {
		return fmt.Errorf("snap: verify asymmetric: %w", err)
	}
	digest := sha256.Sum256([]byte(stringToSign))
	if err := rsa.VerifyPKCS1v15(rsaPub, crypto.SHA256, digest[:], sig); err != nil {
		return fmt.Errorf("snap: verify asymmetric: %w", err)
	}
	return nil
}

// BuildStringToSignAccessToken builds the stringToSign for a SNAP access
// token request: clientID + "|" + timestamp.
func BuildStringToSignAccessToken(clientID, timestamp string) string {
	return clientID + "|" + timestamp
}

// BuildStringToSignTransaction builds the stringToSign for a SNAP
// transaction request from the bytes sent on the wire. The body is minified
// first, as the standard requires (Lowercase(HexEncode(SHA-256(minify(RequestBody))))),
// so whitespace outside JSON strings does not change the signature; a body
// that is not JSON is hashed as is, and an empty body hashes the empty string.
//
// symmetric formula:  HTTPMethod:EndpointUrl:AccessToken:HexSHA256(minify(body)):TimeStamp
// asymmetric formula: HTTPMethod:EndpointUrl:HexSHA256(minify(body)):TimeStamp
func BuildStringToSignTransaction(method, endpointURL, accessToken string, body []byte, timestamp string, symmetric bool) string {
	bodyHash := sha256.Sum256(minify(body))
	bodyHashHex := hex.EncodeToString(bodyHash[:])
	if symmetric {
		return method + ":" + endpointURL + ":" + accessToken + ":" + bodyHashHex + ":" + timestamp
	}
	return method + ":" + endpointURL + ":" + bodyHashHex + ":" + timestamp
}

// ParseRSAPrivateKeyPEM parses a PKCS#1 or PKCS#8 PEM-encoded RSA private
// key. It is a convenience helper; callers that already hold a
// crypto.Signer (e.g. from an HSM/KMS) do not need it.
func ParseRSAPrivateKeyPEM(pemBytes []byte) (crypto.Signer, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("snap: parse rsa private key: %w", ErrNoPEMBlock)
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("snap: parse rsa private key: %w", err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("snap: parse rsa private key: %w", ErrNotRSAKey)
	}
	return rsaKey, nil
}

// minify removes insignificant whitespace from a JSON body (json.Compact).
func minify(body []byte) []byte {
	var b bytes.Buffer
	if err := json.Compact(&b, body); err != nil {
		return body
	}
	return b.Bytes()
}
