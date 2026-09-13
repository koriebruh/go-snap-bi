package transferkredit

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	snap "github.com/koriebruh/go-snap-bi"
	"github.com/koriebruh/go-snap-bi/internal/snaptest"
)

func TestAccountBindingInquiry_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2000800",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"2020102977770000000009",
   "partnerReferenceNo":"2020102900000000000001",
   "accountCurrency":"IDR",
   "accountName":"Alen Miucic",
   "accountNo":"11231271284140",
   "accountTransactionLimit":"1000000",
   "endDatePeriod":"2022-05-21",
   "startDatePeriod":"2020-05-21",
   "additionalInfo":{"channel":"mobilephone"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-account-inquiry"
	tr := &snap.Transport{}
	resp, err := AccountBindingInquiry(context.Background(), tr, hb, AccountBindingInquiryRequest{})
	if err != nil {
		t.Fatalf("AccountBindingInquiry() error = %v", err)
	}

	want := AccountBindingInquiryResponse{
		ResponseCode:            "2000800",
		ResponseMessage:         "Request has been processed successfully",
		ReferenceNo:             "2020102977770000000009",
		PartnerReferenceNo:      "2020102900000000000001",
		AccountCurrency:         "IDR",
		AccountName:             "Alen Miucic",
		AccountNo:               "11231271284140",
		AccountTransactionLimit: json.RawMessage(`"1000000"`),
		EndDatePeriod:           "2022-05-21",
		StartDatePeriod:         "2020-05-21",
		AdditionalInfo:          json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("AccountBindingInquiry() = %+v, want %+v", resp, want)
	}
}

// TestAccountBindingInquiry_TransactionLimitAcceptsEitherWireShape is the
// regression test for a santa-loop finding: the Guides tab labels
// accountTransactionLimit "Numeric", and while this portal's one worked
// example renders it quoted, SNAP is a multi-PJP standard and one example
// is thin evidence about what every issuer sends. A plain string field
// would hard-fail the entire decode (discarding AccountNo/AccountName/etc.
// too) for an unquoted numeric value. json.RawMessage tolerates either
// shape — same fix, same reasoning, as AccountCreationResponse.APIKey.
func TestAccountBindingInquiry_TransactionLimitAcceptsEitherWireShape(t *testing.T) {
	tests := []struct {
		name  string
		limit string
	}{
		{"quoted string", `"1000000"`},
		{"unquoted number", `1000000`},
		// json null decodes to a non-nil 4-byte RawMessage("null"), distinct
		// from an absent key (which decodes to nil) — worth pinning
		// explicitly per the same convention as the APIKey regression test.
		{"json null", `null`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"responseCode":"2000800","responseMessage":"ok","accountNo":"11231271284140","accountTransactionLimit":`+tt.limit+`}`)
			}))
			defer server.Close()

			hb := snaptest.TestHeaderBuilder(server.URL)
			hb.EndpointURL = server.URL + "/v1.0/registration-account-inquiry"
			tr := &snap.Transport{}
			resp, err := AccountBindingInquiry(context.Background(), tr, hb, AccountBindingInquiryRequest{})
			if err != nil {
				t.Fatalf("AccountBindingInquiry() error = %v, want nil — accountTransactionLimit shape must not break the whole decode", err)
			}
			if resp.AccountNo != "11231271284140" {
				t.Errorf("AccountNo = %q, want it to survive regardless of accountTransactionLimit's shape", resp.AccountNo)
			}
			if string(resp.AccountTransactionLimit) != tt.limit {
				t.Errorf("AccountTransactionLimit = %s, want raw bytes %s", resp.AccountTransactionLimit, tt.limit)
			}
		})
	}
}

func TestAccountBindingInquiry_RequestBodyRoundTrips(t *testing.T) {
	var mu sync.Mutex
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		mu.Lock()
		gotBody = b
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"responseCode":"2000800","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-account-inquiry"
	tr := &snap.Transport{}
	req := AccountBindingInquiryRequest{
		PartnerReferenceNo: "ref-1",
		AdditionalInfo:     json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if _, err := AccountBindingInquiry(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("AccountBindingInquiry() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["partnerReferenceNo"] != "ref-1" {
		t.Errorf(`wire body["partnerReferenceNo"] = %v, want "ref-1"`, got["partnerReferenceNo"])
	}
	additionalInfo, ok := got["additionalInfo"].(map[string]any)
	if !ok || additionalInfo["channel"] != "mobilephone" {
		t.Errorf(`wire body["additionalInfo"] = %v, want {"channel":"mobilephone"}`, got["additionalInfo"])
	}
}

func TestAccountBindingInquiry_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4000800","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-account-inquiry"
	tr := &snap.Transport{}
	_, err := AccountBindingInquiry(context.Background(), tr, hb, AccountBindingInquiryRequest{})
	if err == nil {
		t.Fatal("AccountBindingInquiry() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("AccountBindingInquiry() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestAccountBindingInquiry_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2000800","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-account-inquiry"
	tr := &snap.Transport{}
	resp, err := AccountBindingInquiry(context.Background(), tr, hb, AccountBindingInquiryRequest{})
	if err == nil {
		t.Fatalf("AccountBindingInquiry() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("AccountBindingInquiry() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestAccountBindingInquiry_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"accountNo":"11231271284140"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-account-inquiry"
	tr := &snap.Transport{}
	resp, err := AccountBindingInquiry(context.Background(), tr, hb, AccountBindingInquiryRequest{})
	if err == nil {
		t.Fatalf("AccountBindingInquiry() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
