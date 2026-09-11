package snap

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
)

func TestCardRegistrationUnbinding_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2000500",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"2020102977770000000009",
   "partnerReferenceNo":"2020102900000000000001",
   "message":"Card unbinding successful",
   "customerId":"ae75e364134cdb2c7a4159106e38ca6b761983859dbv1",
   "unsubscribeDate":"2020-12-17T13:50:04+07:00",
   "additionalInfo":{"channel":"mobilephone"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-card-unbind"
	tr := &Transport{}
	resp, err := CardRegistrationUnbinding(context.Background(), tr, hb, CardRegistrationUnbindingRequest{Token: "g4JeIz43jfjVvAvNxswe56"})
	if err != nil {
		t.Fatalf("CardRegistrationUnbinding() error = %v", err)
	}

	want := CardRegistrationUnbindingResponse{
		ResponseCode:       "2000500",
		ResponseMessage:    "Request has been processed successfully",
		ReferenceNo:        "2020102977770000000009",
		PartnerReferenceNo: "2020102900000000000001",
		Message:            "Card unbinding successful",
		CustomerID:         "ae75e364134cdb2c7a4159106e38ca6b761983859dbv1",
		UnsubscribeDate:    "2020-12-17T13:50:04+07:00",
		AdditionalInfo:     json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("CardRegistrationUnbinding() = %+v, want %+v", resp, want)
	}
}

func TestCardRegistrationUnbinding_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2000500","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-card-unbind"
	tr := &Transport{}
	req := CardRegistrationUnbindingRequest{
		PartnerReferenceNo: "2020102900000000000001",
		Token:              "g4JeIz43jfjVvAvNxswe56",
		BankCardNo:         "2123123123125356",
		Type:               "Unsubscribe",
		Part:               "00007100010926",
		MerchantID:         "00007100010926",
		SubMerchantID:      "23489182303312",
		TerminalID:         "310928924949487",
		TokenRequestorID:   "7127425327776087324915228",
		JourneyID:          "20190329175623MTISTORE",
		TransactionDate:    "2020-12-17T13:50:00+07:00",
		AdditionalInfo:     json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if _, err := CardRegistrationUnbinding(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("CardRegistrationUnbinding() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["token"] != "g4JeIz43jfjVvAvNxswe56" {
		t.Errorf(`wire body["token"] = %v, want "g4JeIz43jfjVvAvNxswe56"`, got["token"])
	}
	if got["part"] != "00007100010926" {
		t.Errorf(`wire body["part"] = %v, want "00007100010926"`, got["part"])
	}
	additionalInfo, ok := got["additionalInfo"].(map[string]any)
	if !ok || additionalInfo["channel"] != "mobilephone" {
		t.Errorf(`wire body["additionalInfo"] = %v, want {"channel":"mobilephone"}`, got["additionalInfo"])
	}
}

// TestCardRegistrationUnbinding_TokenAlwaysSerialized pins that Token —
// the one request field without omitempty — is always present on the
// wire, even as "", mirroring TestAccountBinding_MerchantIDAlwaysSerialized.
func TestCardRegistrationUnbinding_TokenAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2000500","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-card-unbind"
	tr := &Transport{}
	if _, err := CardRegistrationUnbinding(context.Background(), tr, hb, CardRegistrationUnbindingRequest{}); err != nil {
		t.Fatalf("CardRegistrationUnbinding() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	token, ok := got["token"]
	if !ok {
		t.Fatal(`wire body missing "token" key; Token lacks omitempty and must always be present, even as ""`)
	}
	if token != "" {
		t.Errorf(`wire body["token"] = %v, want ""`, token)
	}
}

func TestCardRegistrationUnbinding_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4000500","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-card-unbind"
	tr := &Transport{}
	_, err := CardRegistrationUnbinding(context.Background(), tr, hb, CardRegistrationUnbindingRequest{Token: "tok"})
	if err == nil {
		t.Fatal("CardRegistrationUnbinding() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("CardRegistrationUnbinding() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestCardRegistrationUnbinding_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2000500","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-card-unbind"
	tr := &Transport{}
	resp, err := CardRegistrationUnbinding(context.Background(), tr, hb, CardRegistrationUnbindingRequest{Token: "tok"})
	if err == nil {
		t.Fatalf("CardRegistrationUnbinding() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("CardRegistrationUnbinding() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestCardRegistrationUnbinding_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"customerId":"cust"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-card-unbind"
	tr := &Transport{}
	resp, err := CardRegistrationUnbinding(context.Background(), tr, hb, CardRegistrationUnbindingRequest{Token: "tok"})
	if err == nil {
		t.Fatalf("CardRegistrationUnbinding() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
