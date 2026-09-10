package snap

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestTransportDo_ParsesEnvelopeAndPreservesRaw(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"responseCode":"2000000","responseMessage":"Successful","referenceNo":"REF-123","amount":"1000.00"}`)
	}))
	defer server.Close()

	hb := HeaderBuilder{
		Method:      http.MethodPost,
		EndpointURL: server.URL + "/service/endpoint",
		Body:        []byte(`{"foo":"bar"}`),
		AccessToken: "access-token",
		ClientKey:   "client-key",
		PartnerID:   "partner-id",
		ExternalID:  "external-id",
		ChannelID:   "channel-id",

		Symmetric:    true,
		ClientSecret: "client-secret",
	}

	tr := &Transport{}
	env, err := tr.Do(context.Background(), hb)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if env.ResponseCode != "2000000" {
		t.Errorf("ResponseCode = %q, want %q", env.ResponseCode, "2000000")
	}
	if env.ResponseMessage != "Successful" {
		t.Errorf("ResponseMessage = %q, want %q", env.ResponseMessage, "Successful")
	}

	var extra struct {
		ReferenceNo string `json:"referenceNo"`
		Amount      string `json:"amount"`
	}
	if err := json.Unmarshal(env.Raw, &extra); err != nil {
		t.Fatalf("unmarshal Raw: %v", err)
	}
	if extra.ReferenceNo != "REF-123" || extra.Amount != "1000.00" {
		t.Errorf("extra fields from Raw = %+v, want ReferenceNo=REF-123 Amount=1000.00", extra)
	}
}

func TestTransportDo_SendsSignedHeaders(t *testing.T) {
	var mu sync.Mutex
	var gotMethod, gotTimestamp, gotSignature string
	var gotBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		gotMethod = r.Method
		gotTimestamp = r.Header.Get("X-Timestamp")
		gotSignature = r.Header.Get("X-Signature")
		gotBody = body
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"responseCode":"2000000","responseMessage":"Successful"}`)
	}))
	defer server.Close()

	const clientSecret = "client-secret"
	hb := HeaderBuilder{
		Method:      http.MethodPost,
		EndpointURL: server.URL + "/service/endpoint",
		Body:        []byte(`{"foo":"bar"}`),
		AccessToken: "access-token",
		ClientKey:   "client-key",
		PartnerID:   "partner-id",
		ExternalID:  "external-id",
		ChannelID:   "channel-id",

		Symmetric:    true,
		ClientSecret: clientSecret,
	}

	tr := &Transport{}
	if _, err := tr.Do(context.Background(), hb); err != nil {
		t.Fatalf("Do() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if gotMethod != http.MethodPost {
		t.Errorf("request method = %q, want POST", gotMethod)
	}
	if gotTimestamp == "" {
		t.Error("X-Timestamp header missing on outgoing request")
	}
	if gotSignature == "" {
		t.Fatal("X-Signature header missing on outgoing request")
	}

	// Recompute the expected stringToSign from what the server actually
	// received and verify the signature against it, so this test fails if
	// the formula (or the symmetric/asymmetric choice) is wrong, not just
	// if the header is empty.
	wantStringToSign := BuildStringToSignTransaction(gotMethod, hb.EndpointURL, hb.AccessToken, gotBody, gotTimestamp, true)
	if !VerifySymmetric(clientSecret, wantStringToSign, gotSignature) {
		t.Errorf("X-Signature does not verify against the recomputed stringToSign %q", wantStringToSign)
	}
}

func TestTransportDo_PropagatesHeaderBuildErrorUnchanged(t *testing.T) {
	hb := HeaderBuilder{
		Method:      http.MethodPost,
		EndpointURL: "https://example.invalid/service/endpoint",
		// ExternalID intentionally left empty: HeaderBuilder.Build must
		// reject this before any request is sent, and Do must return that
		// exact error rather than wrapping it.
		Symmetric:    true,
		ClientSecret: "client-secret",
	}

	tr := &Transport{}
	_, gotErr := tr.Do(context.Background(), hb)
	wantErr := errBuildHeaders(hb)
	if gotErr == nil || gotErr.Error() != wantErr.Error() {
		t.Errorf("Do() error = %v, want unchanged HeaderBuilder.Build() error %v", gotErr, wantErr)
	}
}

// errBuildHeaders mirrors what hb.Build() returns for the malformed hb above,
// used only to assert Do() doesn't alter/wrap that error.
func errBuildHeaders(hb HeaderBuilder) error {
	_, err := hb.Build()
	return err
}

func TestTransportDo_NonTwoXXStatusIsNotAnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"responseCode":"4017301","responseMessage":"Unauthorized. Invalid Signature"}`)
	}))
	defer server.Close()

	hb := HeaderBuilder{
		Method:      http.MethodPost,
		EndpointURL: server.URL + "/service/endpoint",
		Body:        []byte(`{}`),
		AccessToken: "access-token",
		ClientKey:   "client-key",
		PartnerID:   "partner-id",
		ExternalID:  "external-id",
		ChannelID:   "channel-id",

		Symmetric:    true,
		ClientSecret: "client-secret",
	}

	tr := &Transport{}
	env, err := tr.Do(context.Background(), hb)
	if err != nil {
		t.Fatalf("Do() error = %v, want nil (non-2xx is not a transport error)", err)
	}
	if env.ResponseCode != "4017301" {
		t.Errorf("ResponseCode = %q, want %q", env.ResponseCode, "4017301")
	}
	if err := ResponseCodeError(env.ResponseCode); err == nil {
		t.Error("ResponseCodeError(env.ResponseCode) = nil, want non-nil so caller can detect the failure")
	}
}
