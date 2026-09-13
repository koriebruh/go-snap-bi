package registration

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

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-card-bind"
	tr := &snap.Transport{}
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

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-card-bind"
	tr := &snap.Transport{}
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
// json.Marshal accepts every valid-JSON shape and forwards it to the wire
// (whitespace compacted, as encoding/json always does for RawMessage —
// not a caller-visible byte-for-byte passthrough), rather than the
// decode-failure risk TestAccountCreation_APIKeyAcceptsEitherWireShape (a
// response field) guards against.
func TestCardRegistration_LimitAndCardDataAcceptEitherWireShape(t *testing.T) {
	tests := []struct {
		name     string
		limit    string
		cardData string
	}{
		{"quoted string", `"1000000"`, `"encrypted-blob"`},
		{"unquoted number and bare object", `1000000`, `{"k":"v"}`},
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

			hb := snaptest.TestHeaderBuilder(server.URL)
			hb.EndpointURL = server.URL + "/v1.0/registration-card-bind"
			tr := &snap.Transport{}
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
				t.Errorf(`wire body["limit"] = %s, want %s (shape preserved)`, got["limit"], tt.limit)
			}
			if string(got["cardData"]) != tt.cardData {
				t.Errorf(`wire body["cardData"] = %s, want %s (shape preserved)`, got["cardData"], tt.cardData)
			}
		})
	}
}

// TestCardRegistration_NonNumericUnquotedLimitFailsEncode pins one of the
// two failure modes CardRegistrationRequest's doc comment warns about: a
// comma-formatted decimal like "1,000,000" is not valid JSON on its own
// (unlike a bare digit string, which the "unquoted number and bare
// object" case above marshals fine and sends as-is, silently), so
// json.Marshal fails and
// CardRegistration returns an encode error before any request reaches
// the server.
func TestCardRegistration_NonNumericUnquotedLimitFailsEncode(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"responseCode":"2000100","responseMessage":"ok","bankCardToken":"tok"}`)
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-card-bind"
	tr := &snap.Transport{}
	req := CardRegistrationRequest{
		BankCardNo:     "3984029384023984",
		CustIDMerchant: "0012345679504",
		Limit:          json.RawMessage(`1,000,000`),
	}
	_, err := CardRegistration(context.Background(), tr, hb, req)
	if err == nil {
		t.Fatal("CardRegistration() error = nil, want non-nil for a non-numeric unquoted limit")
	}
	if called {
		t.Error("CardRegistration() reached the server despite a marshal failure; want the request never sent")
	}
}

// TestCardRegistration_InvalidCardDataFailsEncode mirrors
// TestCardRegistration_NonNumericUnquotedLimitFailsEncode for CardData:
// this blob starts with a letter, so it is not a valid JSON value token
// on its own, and json.Marshal fails. This is specific to this blob, not
// a property of "base64" in general: a same-shape blob that happened to
// be all digits with no leading zero would marshal fine as a bare
// number instead, silently — the failure/success boundary here is
// "is this string a complete JSON value", not "is this a base64 blob".
func TestCardRegistration_InvalidCardDataFailsEncode(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"responseCode":"2000100","responseMessage":"ok","bankCardToken":"tok"}`)
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-card-bind"
	tr := &snap.Transport{}
	req := CardRegistrationRequest{
		BankCardNo:     "3984029384023984",
		CustIDMerchant: "0012345679504",
		CardData:       json.RawMessage(`UIdFgZi9BhWx9Scbz/YK+abc=`),
	}
	_, err := CardRegistration(context.Background(), tr, hb, req)
	if err == nil {
		t.Fatal("CardRegistration() error = nil, want non-nil for an unquoted base64 cardData blob")
	}
	if called {
		t.Error("CardRegistration() reached the server despite a marshal failure; want the request never sent")
	}
}

// TestCardRegistration_MandatoryFieldsAlwaysSerialized pins that
// BankCardNo and CustIDMerchant — the two request fields without
// omitempty — are always present on the wire, even as "", mirroring
// TestAccountBinding_MerchantIDAlwaysSerialized/
// TestAccountUnbinding_MerchantIDAlwaysSerialized.
func TestCardRegistration_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-card-bind"
	tr := &snap.Transport{}
	if _, err := CardRegistration(context.Background(), tr, hb, CardRegistrationRequest{}); err != nil {
		t.Fatalf("CardRegistration() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	for _, key := range []string{"bankCardNo", "custIdMerchant"} {
		v, ok := got[key]
		if !ok {
			t.Errorf(`wire body missing %q key; want it always present, even as ""`, key)
			continue
		}
		if v != "" {
			t.Errorf(`wire body[%q] = %v, want ""`, key, v)
		}
	}
}

// TestCardRegistration_EmptyLimitOmitsField pins the third documented
// behavior: an empty json.RawMessage is dropped by omitempty, so the
// field is absent from the wire body, not sent as an empty value.
func TestCardRegistration_EmptyLimitOmitsField(t *testing.T) {
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

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-card-bind"
	tr := &snap.Transport{}
	req := CardRegistrationRequest{
		BankCardNo:     "3984029384023984",
		CustIDMerchant: "0012345679504",
		Limit:          json.RawMessage(``),
	}
	if _, err := CardRegistration(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("CardRegistration() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]json.RawMessage
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if _, ok := got["limit"]; ok {
		t.Errorf(`wire body has "limit" key = %s, want key absent for an empty RawMessage`, got["limit"])
	}
}

func TestCardRegistration_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4000100","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-card-bind"
	tr := &snap.Transport{}
	_, err := CardRegistration(context.Background(), tr, hb, CardRegistrationRequest{BankCardNo: "3984029384023984", CustIDMerchant: "0012345679504"})
	if err == nil {
		t.Fatal("CardRegistration() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("CardRegistration() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestCardRegistration_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2000100","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-card-bind"
	tr := &snap.Transport{}
	resp, err := CardRegistration(context.Background(), tr, hb, CardRegistrationRequest{BankCardNo: "3984029384023984", CustIDMerchant: "0012345679504"})
	if err == nil {
		t.Fatalf("CardRegistration() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("CardRegistration() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestCardRegistration_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"bankCardToken":"tok"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-card-bind"
	tr := &snap.Transport{}
	resp, err := CardRegistration(context.Background(), tr, hb, CardRegistrationRequest{BankCardNo: "3984029384023984", CustIDMerchant: "0012345679504"})
	if err == nil {
		t.Fatalf("CardRegistration() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
