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
	if _, err := CardRegistrationInquiry(context.Background(), tr, hb, "abc/def"); err != nil {
		t.Fatalf("CardRegistrationInquiry() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	// r.URL.Path is Go's decoded form (net/http always decodes %2F back to
	// "/" there, even when the wire bytes carried the escaped form); the
	// literal wire bytes are only visible via EscapedPath()/RawPath.
	want := "/v1.0/registration-card-inquiry/custIdMerchant/abc%2Fdef"
	if gotEscapedPath != want {
		t.Errorf("request wire path = %q, want %q (custIDMerchant containing '/' must be percent-encoded on the wire, not split into extra path segments)", gotEscapedPath, want)
	}
}

func TestCardRegistrationInquiry_UsesGETWithNoBodyRegardlessOfCallerHeaderBuilder(t *testing.T) {
	var mu sync.Mutex
	var gotMethod string
	var gotBodyLen int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotMethod = r.Method
		mu.Unlock()
		b := make([]byte, 1)
		n, _ := r.Body.Read(b)
		mu.Lock()
		gotBodyLen = n
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
	if gotBodyLen != 0 {
		t.Errorf("request body was non-empty, want none even though the caller's HeaderBuilder had a Body set")
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
