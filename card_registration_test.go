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

func TestCardRegistration_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2000100",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"2020102977770000000009",
   "partnerReferenceNo":"2020102900000000000001",
   "bankCardToken":"6d7963617264746f6b656e",
   "chargeToken":"abcd63617264746f6b656e",
   "randomString":"g4BoEz43jfjVvAvN",
   "tokenExpiryTime":"2020-12-17T11:00:00+07:00",
   "additionalInfo":{"channel":"mobilephone"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-card-bind"
	tr := &Transport{}
	resp, err := CardRegistration(context.Background(), tr, hb, CardRegistrationRequest{BankCardNo: "3984029384023984", CustIDMerchant: "0012345679504"})
	if err != nil {
		t.Fatalf("CardRegistration() error = %v", err)
	}

	want := CardRegistrationResponse{
		ResponseCode:       "2000100",
		ResponseMessage:    "Request has been processed successfully",
		ReferenceNo:        "2020102977770000000009",
		PartnerReferenceNo: "2020102900000000000001",
		BankCardToken:      "6d7963617264746f6b656e",
		ChargeToken:        "abcd63617264746f6b656e",
		RandomString:       "g4BoEz43jfjVvAvN",
		TokenExpiryTime:    "2020-12-17T11:00:00+07:00",
		AdditionalInfo:     json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("CardRegistration() = %+v, want %+v", resp, want)
	}
}

func TestCardRegistration_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2000100","responseMessage":"ok","bankCardToken":"tok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-card-bind"
	tr := &Transport{}
	req := CardRegistrationRequest{
		PartnerReferenceNo: "ref-1",
		CardData:           json.RawMessage(`"encrypted-blob"`),
		BankCardNo:         "3984029384023984",
		CustIDMerchant:     "0012345679504",
		Limit:              json.RawMessage(`"1000000"`),
		AdditionalInfo:     json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if _, err := CardRegistration(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("CardRegistration() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["bankCardNo"] != "3984029384023984" {
		t.Errorf(`wire body["bankCardNo"] = %v, want "3984029384023984"`, got["bankCardNo"])
	}
	if got["custIdMerchant"] != "0012345679504" {
		t.Errorf(`wire body["custIdMerchant"] = %v, want "0012345679504"`, got["custIdMerchant"])
	}
	if got["cardData"] != "encrypted-blob" {
		t.Errorf(`wire body["cardData"] = %v, want "encrypted-blob"`, got["cardData"])
	}
	if got["limit"] != "1000000" {
		t.Errorf(`wire body["limit"] = %v, want "1000000"`, got["limit"])
	}
	additionalInfo, ok := got["additionalInfo"].(map[string]any)
	if !ok || additionalInfo["channel"] != "mobilephone" {
		t.Errorf(`wire body["additionalInfo"] = %v, want {"channel":"mobilephone"}`, got["additionalInfo"])
	}
}

// TestCardRegistration_LimitAndCardDataAcceptEitherWireShape is the
// regression test for CardData/Limit: the Guides tab labels them
// "Encrypted Object"/"decimal", both of which permit a non-string JSON
// representation, even though the portal's worked example quotes both.
// Both are request-only fields, so the guarantee under test is that
// json.Marshal accepts every shape and sends it verbatim — unlike
// TestAccountCreation_APIKeyAcceptsEitherWireShape (a response field,
// where the risk is decode failure), the risk here is a caller-supplied
// shape reaching the wire unexamined, so this test asserts the captured
// request body byte-for-byte rather than only that the call succeeded.
func TestCardRegistration_LimitAndCardDataAcceptEitherWireShape(t *testing.T) {
	tests := []struct {
		name     string
		limit    string
		cardData string
	}{
		{"quoted string", `"1000000"`, `"encrypted-blob"`},
		{"unquoted number", `1000000`, `{"k":"v"}`},
		{"json null", `null`, `null`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
				_, _ = io.WriteString(w, `{"responseCode":"2000100","responseMessage":"ok","bankCardToken":"tok"}`)
			}))
			defer server.Close()

			hb := testHeaderBuilder(server.URL)
			hb.EndpointURL = server.URL + "/v1.0/registration-card-bind"
			tr := &Transport{}
			req := CardRegistrationRequest{
				BankCardNo:     "3984029384023984",
				CustIDMerchant: "0012345679504",
				Limit:          json.RawMessage(tt.limit),
				CardData:       json.RawMessage(tt.cardData),
			}
			if _, err := CardRegistration(context.Background(), tr, hb, req); err != nil {
				t.Fatalf("CardRegistration() error = %v, want nil — limit/cardData shape must not break the call", err)
			}

			mu.Lock()
			defer mu.Unlock()
			var got map[string]json.RawMessage
			if err := json.Unmarshal(gotBody, &got); err != nil {
				t.Fatalf("decode request body the server received: %v", err)
			}
			if string(got["limit"]) != tt.limit {
				t.Errorf(`wire body["limit"] = %s, want %s (exact shape sent as given, not normalized)`, got["limit"], tt.limit)
			}
			if string(got["cardData"]) != tt.cardData {
				t.Errorf(`wire body["cardData"] = %s, want %s (exact shape sent as given, not normalized)`, got["cardData"], tt.cardData)
			}
		})
	}
}

func TestCardRegistration_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4000100","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-card-bind"
	tr := &Transport{}
	_, err := CardRegistration(context.Background(), tr, hb, CardRegistrationRequest{BankCardNo: "3984029384023984", CustIDMerchant: "0012345679504"})
	if err == nil {
		t.Fatal("CardRegistration() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("CardRegistration() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestCardRegistration_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2000100","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-card-bind"
	tr := &Transport{}
	resp, err := CardRegistration(context.Background(), tr, hb, CardRegistrationRequest{BankCardNo: "3984029384023984", CustIDMerchant: "0012345679504"})
	if err == nil {
		t.Fatalf("CardRegistration() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("CardRegistration() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestCardRegistration_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"bankCardToken":"tok"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-card-bind"
	tr := &Transport{}
	resp, err := CardRegistration(context.Background(), tr, hb, CardRegistrationRequest{BankCardNo: "3984029384023984", CustIDMerchant: "0012345679504"})
	if err == nil {
		t.Fatalf("CardRegistration() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
