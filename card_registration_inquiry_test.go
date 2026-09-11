package snap

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
)

func TestCardRegistrationInquiry_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2000300",
   "responseMessage":"Request has been processed successfully",
   "accountList":[
      {"accountData":{
         "accountId":"F8FP2WQWEATXFP8K",
         "createdDate":"2018-12-17T11:59:06+07:00",
         "credentialNo":"************0750",
         "credentialType":"DC",
         "maxLimit":"800000",
         "status":"ACT"
      }}
   ],
   "additionalInfo":{"channel":"mobilephone"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-card-inquiry"
	tr := &Transport{}
	resp, err := CardRegistrationInquiry(context.Background(), tr, hb, "8a95f0026d2860f301")
	if err != nil {
		t.Fatalf("CardRegistrationInquiry() error = %v", err)
	}

	want := CardRegistrationInquiryResponse{
		ResponseCode:    "2000300",
		ResponseMessage: "Request has been processed successfully",
		AccountList: []CardRegistrationInquiryAccount{
			{
				AccountData: CardRegistrationInquiryAccountData{
					AccountID:      "F8FP2WQWEATXFP8K",
					CreatedDate:    "2018-12-17T11:59:06+07:00",
					CredentialNo:   "************0750",
					CredentialType: "DC",
					MaxLimit:       "800000",
					Status:         "ACT",
				},
			},
		},
		AdditionalInfo: json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("CardRegistrationInquiry() = %+v, want %+v", resp, want)
	}
}

func TestCardRegistrationInquiry_URLConstruction(t *testing.T) {
	var mu sync.Mutex
	var gotEscapedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotEscapedPath = r.URL.EscapedPath()
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"responseCode":"2000300","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-card-inquiry"
	tr := &Transport{}
	if _, err := CardRegistrationInquiry(context.Background(), tr, hb, "cust_ID-1"); err != nil {
		t.Fatalf("CardRegistrationInquiry() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	want := "/v1.0/registration-card-inquiry/custIdMerchant/cust_ID-1"
	if gotEscapedPath != want {
		t.Errorf("request wire path = %q, want %q", gotEscapedPath, want)
	}
}

// TestCardRegistrationInquiry_RejectsQueryOrFragmentEndpointURL is the
// regression test for a santa-loop finding: string concatenation onto an
// EndpointURL carrying a fragment silently dropped custIDMerchant from
// the transmitted request (net/http never sends a URL fragment at all),
// and one carrying a query string moved the segment into RawQuery
// instead of Path — neither produced an error, both produced a request
// to the wrong resource. Parsing hb.EndpointURL and rejecting either
// case outright closes both silently-wrong outcomes.
func TestCardRegistrationInquiry_RejectsQueryOrFragmentEndpointURL(t *testing.T) {
	for _, suffix := range []string{"?trace=1", "#frag"} {
		t.Run(suffix, func(t *testing.T) {
			called := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"responseCode":"2000300","responseMessage":"ok"}`))
			}))
			defer server.Close()

			hb := testHeaderBuilder(server.URL)
			hb.EndpointURL = server.URL + "/v1.0/registration-card-inquiry" + suffix
			tr := &Transport{}
			_, err := CardRegistrationInquiry(context.Background(), tr, hb, "cust-1")
			if err == nil {
				t.Fatalf("CardRegistrationInquiry() error = nil, want non-nil for EndpointURL ending in %q", suffix)
			}
			if called {
				t.Error("CardRegistrationInquiry() reached the server despite an invalid EndpointURL; want the request never sent")
			}
		})
	}
}

func TestCardRegistrationInquiry_UsesGETWithNoBodyRegardlessOfCallerHeaderBuilder(t *testing.T) {
	var mu sync.Mutex
	var gotMethod string
	var gotContentLength int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotMethod = r.Method
		gotContentLength = r.ContentLength
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"responseCode":"2000300","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL) // Method: http.MethodPost, by default
	hb.EndpointURL = server.URL + "/v1.0/registration-card-inquiry"
	hb.Body = []byte(`{"should":"be ignored"}`)
	tr := &Transport{}
	if _, err := CardRegistrationInquiry(context.Background(), tr, hb, "cust-1"); err != nil {
		t.Fatalf("CardRegistrationInquiry() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if gotMethod != http.MethodGet {
		t.Errorf("request method = %q, want %q even though the caller's HeaderBuilder had Method=POST", gotMethod, http.MethodGet)
	}
	if gotContentLength != 0 {
		t.Errorf("request Content-Length = %d, want 0 even though the caller's HeaderBuilder had a Body set", gotContentLength)
	}
}

// TestCardRegistrationInquiry_RejectsInvalidCustIDMerchant is the
// regression test for two review rounds: a first pass (go-review) caught
// that url.PathEscape doesn't escape "." or ".."; a second pass
// (santa-loop round 1) caught that a three-value denylist for "" / "." /
// ".." still lets a multi-segment value like "x/../../other" through
// (it reaches the wire as one escaped segment, "x%2F..%2F..%2Fother",
// which a decode-then-normalize intermediary could unfold back into real
// path boundaries) and doesn't cover invalid UTF-8 or Unicode dot
// lookalikes either. custIDMerchantPattern's allowlist closes all of
// these in one guard instead of enumerating each variant.
func TestCardRegistrationInquiry_RejectsInvalidCustIDMerchant(t *testing.T) {
	values := []string{
		"",
		".",
		"..",
		"abc/def",
		"x/../../other",
		"..\\",
		"%2e%2e",
		"\xc0\xae\xc0\xae", // overlong UTF-8 encoding of ".."
		"cust․․",           // U+2024 ONE DOT LEADER, an NFKC "." lookalike
		"cust?trace=1",
	}
	for _, custIDMerchant := range values {
		t.Run(custIDMerchant, func(t *testing.T) {
			var mu sync.Mutex
			called := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				called = true
				mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"responseCode":"2000300","responseMessage":"ok"}`))
			}))
			defer server.Close()

			hb := testHeaderBuilder(server.URL)
			hb.EndpointURL = server.URL + "/v1.0/registration-card-inquiry"
			tr := &Transport{}
			_, err := CardRegistrationInquiry(context.Background(), tr, hb, custIDMerchant)
			if err == nil {
				t.Fatalf("CardRegistrationInquiry() error = nil, want non-nil for custIDMerchant = %q", custIDMerchant)
			}
			mu.Lock()
			defer mu.Unlock()
			if called {
				t.Error("CardRegistrationInquiry() reached the server despite an invalid custIDMerchant; want the request never sent")
			}
		})
	}
}

// TestCardRegistrationInquiry_TrimsTrailingSlashOnEndpointURL guards
// against a caller-supplied EndpointURL ending in "/" producing
// ".../custIdMerchant//cust-1" (a double slash a slash-merging gateway
// could rewrite after the signature was already computed over the
// unmerged form).
func TestCardRegistrationInquiry_TrimsTrailingSlashOnEndpointURL(t *testing.T) {
	var mu sync.Mutex
	var gotEscapedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotEscapedPath = r.URL.EscapedPath()
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"responseCode":"2000300","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-card-inquiry/"
	tr := &Transport{}
	if _, err := CardRegistrationInquiry(context.Background(), tr, hb, "cust-1"); err != nil {
		t.Fatalf("CardRegistrationInquiry() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	want := "/v1.0/registration-card-inquiry/custIdMerchant/cust-1"
	if gotEscapedPath != want {
		t.Errorf("request wire path = %q, want %q (a trailing slash on the caller's EndpointURL must not produce a double slash)", gotEscapedPath, want)
	}
}

// TestCardRegistrationInquiry_SignatureCoversGETRequest closes the one
// part of this GET-shaped request no other test exercises: that the
// signature is actually computed over method=GET, an empty body, and the
// final escaped-path URL — not left over from some other request shape.
func TestCardRegistrationInquiry_SignatureCoversGETRequest(t *testing.T) {
	var mu sync.Mutex
	var gotTimestamp, gotSignature string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotTimestamp = r.Header.Get("X-TIMESTAMP")
		gotSignature = r.Header.Get("X-SIGNATURE")
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"responseCode":"2000300","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-card-inquiry"
	tr := &Transport{}
	if _, err := CardRegistrationInquiry(context.Background(), tr, hb, "cust-1"); err != nil {
		t.Fatalf("CardRegistrationInquiry() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	wantEndpoint := server.URL + "/v1.0/registration-card-inquiry/custIdMerchant/cust-1"
	stringToSign := BuildStringToSignTransaction(http.MethodGet, wantEndpoint, hb.AccessToken, nil, gotTimestamp, hb.Symmetric)
	wantSignature := SignSymmetric(hb.ClientSecret, stringToSign)
	if gotSignature != wantSignature {
		t.Errorf("X-SIGNATURE = %q, want %q (computed from method=GET, a nil body, and the escaped custIdMerchant path)", gotSignature, wantSignature)
	}
}

func TestCardRegistrationInquiry_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4000300","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-card-inquiry"
	tr := &Transport{}
	_, err := CardRegistrationInquiry(context.Background(), tr, hb, "cust-1")
	if err == nil {
		t.Fatal("CardRegistrationInquiry() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("CardRegistrationInquiry() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestCardRegistrationInquiry_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2000300","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-card-inquiry"
	tr := &Transport{}
	resp, err := CardRegistrationInquiry(context.Background(), tr, hb, "cust-1")
	if err == nil {
		t.Fatalf("CardRegistrationInquiry() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("CardRegistrationInquiry() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestCardRegistrationInquiry_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"accountList":[]}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-card-inquiry"
	tr := &Transport{}
	resp, err := CardRegistrationInquiry(context.Background(), tr, hb, "cust-1")
	if err == nil {
		t.Fatalf("CardRegistrationInquiry() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
