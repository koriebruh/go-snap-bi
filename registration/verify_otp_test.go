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

func TestVerifyOTP_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2000400",
   "responseMessage":"Request has been processed successfully",
   "originalReferenceNo":"2020102977770000000009",
   "originalPartnerReferenceNo":"2020102900000000000001",
   "accountNo":"12345678910",
   "bankCardToken":"6d7963617264746f6b656e",
   "cardPan":"2123123123125356",
   "customerId":"afhw6d7963617264746f6b656e963617264746f6b656e",
   "email":"john.doe@email.com",
   "expiredDatetime":"2019-02-24T14:12:25.871+07:00",
   "expiryDate":"1219",
   "identificationNo":"2020102020202000011001",
   "linkageToken":"xswe56",
   "phoneNo":"0899345678864332",
   "qParamsURL":"https://setPin",
   "qParams":{"action":"otpLinkage"},
   "sendOtpFlag":"YES",
   "subscribeDatetime":"2017-02-24T14:12:25.871+07:00",
   "tokenExpiryTime":"2017-02-24T14:12:25.871+07:00",
   "transactionTimestamp":"g4BoEz43jfjVvAvN",
   "additionalInfo":{"channel":"mobilephone"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/otp-verification"
	tr := &snap.Transport{}
	resp, err := VerifyOTP(context.Background(), tr, hb, VerifyOTPRequest{})
	if err != nil {
		t.Fatalf("VerifyOTP() error = %v", err)
	}

	want := VerifyOTPResponse{
		ResponseCode:               "2000400",
		ResponseMessage:            "Request has been processed successfully",
		OriginalReferenceNo:        "2020102977770000000009",
		OriginalPartnerReferenceNo: "2020102900000000000001",
		AccountNo:                  "12345678910",
		BankCardToken:              "6d7963617264746f6b656e",
		CardPan:                    "2123123123125356",
		CustomerID:                 "afhw6d7963617264746f6b656e963617264746f6b656e",
		Email:                      "john.doe@email.com",
		ExpiredDatetime:            "2019-02-24T14:12:25.871+07:00",
		ExpiryDate:                 "1219",
		IdentificationNo:           "2020102020202000011001",
		LinkageToken:               "xswe56",
		PhoneNo:                    "0899345678864332",
		QParamsURL:                 "https://setPin",
		QParams:                    json.RawMessage(`{"action":"otpLinkage"}`),
		SendOTPFlag:                "YES",
		SubscribeDatetime:          "2017-02-24T14:12:25.871+07:00",
		TokenExpiryTime:            "2017-02-24T14:12:25.871+07:00",
		TransactionTimestamp:       "g4BoEz43jfjVvAvN",
		AdditionalInfo:             json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("VerifyOTP() = %+v, want %+v", resp, want)
	}
}

// TestVerifyOTPResponse_QParamsURLMarshalsWithDocumentedCasing pins the
// deliberate "qParamsURL" (capital URL) wire tag on the marshal side: a
// decode-only test can't catch a regression to "qParamsUrl" here, since
// encoding/json's case-insensitive fallback key matching would still
// populate the field correctly either way.
func TestVerifyOTPResponse_QParamsURLMarshalsWithDocumentedCasing(t *testing.T) {
	b, err := json.Marshal(VerifyOTPResponse{ResponseCode: "2000400", ResponseMessage: "ok", QParamsURL: "https://setPin"})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled response: %v", err)
	}
	if _, ok := got["qParamsURL"]; !ok {
		t.Errorf(`marshaled response missing "qParamsURL" key; got keys: %v`, got)
	}
}

func TestVerifyOTP_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2000400","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/otp-verification"
	tr := &snap.Transport{}
	req := VerifyOTPRequest{
		OriginalPartnerReferenceNo: "2020102900000000000001",
		OriginalReferenceNo:        "2020102977770000000009",
		Action:                     "otpLinkage",
		MerchantID:                 "00001",
		OTP:                        "12345678",
		ChargeToken:                "TOK_TKNCPPPHUVL3IJVAXZI5GG4WBEC77YZ6::ADVQ",
		Type:                       "Subscribe",
		AdditionalInfo:             json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if _, err := VerifyOTP(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("VerifyOTP() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["otp"] != "12345678" {
		t.Errorf(`wire body["otp"] = %v, want "12345678"`, got["otp"])
	}
	if got["originalReferenceNo"] != "2020102977770000000009" {
		t.Errorf(`wire body["originalReferenceNo"] = %v, want "2020102977770000000009"`, got["originalReferenceNo"])
	}
	if got["originalPartnerReferenceNo"] != "2020102900000000000001" {
		t.Errorf(`wire body["originalPartnerReferenceNo"] = %v, want "2020102900000000000001"`, got["originalPartnerReferenceNo"])
	}
	if got["chargeToken"] != "TOK_TKNCPPPHUVL3IJVAXZI5GG4WBEC77YZ6::ADVQ" {
		t.Errorf(`wire body["chargeToken"] = %v, want the test's charge token`, got["chargeToken"])
	}
	additionalInfo, ok := got["additionalInfo"].(map[string]any)
	if !ok || additionalInfo["channel"] != "mobilephone" {
		t.Errorf(`wire body["additionalInfo"] = %v, want {"channel":"mobilephone"}`, got["additionalInfo"])
	}
}

func TestVerifyOTP_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4000400","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/otp-verification"
	tr := &snap.Transport{}
	_, err := VerifyOTP(context.Background(), tr, hb, VerifyOTPRequest{})
	if err == nil {
		t.Fatal("VerifyOTP() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("VerifyOTP() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestVerifyOTP_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2000400","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/otp-verification"
	tr := &snap.Transport{}
	resp, err := VerifyOTP(context.Background(), tr, hb, VerifyOTPRequest{})
	if err == nil {
		t.Fatalf("VerifyOTP() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("VerifyOTP() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestVerifyOTP_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"accountNo":"12345678910"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/otp-verification"
	tr := &snap.Transport{}
	resp, err := VerifyOTP(context.Background(), tr, hb, VerifyOTPRequest{})
	if err == nil {
		t.Fatalf("VerifyOTP() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
