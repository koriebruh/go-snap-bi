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

func TestAccountBinding_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2000700",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"2020102977770000000009",
   "partnerReferenceNo":"2020102900000000000001",
   "accountToken":"acct-token-123",
   "accessTokenInfo":{"accessToken":"at-1","expiresIn":"2026-01-01T00:00:00Z","refreshToken":"rt-1","reExpiresIn":"2026-02-01T00:00:00Z","tokenStatus":"active"},
   "linkId":"link-1",
   "nextAction":"redirect",
   "linkageToken":"linkage-1",
   "params":{"pinField":"value"},
   "pinWebViewUrl":"https://example.com/pin",
   "redirectToDeeplink":"app://deeplink",
   "redirectUrl":"https://example.com/redirect",
   "userInfo":{"publicUserId":"user-123"},
   "additionalInfo":{"channel":"mobilephone"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-account-binding"
	tr := &snap.Transport{}
	resp, err := AccountBinding(context.Background(), tr, hb, AccountBindingRequest{MerchantID: "merchant-1"})
	if err != nil {
		t.Fatalf("AccountBinding() error = %v", err)
	}

	want := AccountBindingResponse{
		ResponseCode:       "2000700",
		ResponseMessage:    "Request has been processed successfully",
		ReferenceNo:        "2020102977770000000009",
		PartnerReferenceNo: "2020102900000000000001",
		AccountToken:       "acct-token-123",
		AccessTokenInfo: &BindingAccessTokenInfo{
			AccessToken:  "at-1",
			ExpiresIn:    "2026-01-01T00:00:00Z",
			RefreshToken: "rt-1",
			ReExpiresIn:  "2026-02-01T00:00:00Z",
			TokenStatus:  "active",
		},
		LinkID:             "link-1",
		NextAction:         "redirect",
		LinkageToken:       "linkage-1",
		Params:             json.RawMessage(`{"pinField":"value"}`),
		PinWebViewURL:      "https://example.com/pin",
		RedirectToDeeplink: "app://deeplink",
		RedirectURL:        "https://example.com/redirect",
		UserInfo:           &BindingUserInfo{PublicUserID: "user-123"},
		AdditionalInfo:     json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("AccountBinding() = %+v, want %+v", resp, want)
	}
}

func TestAccountBinding_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2000700","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-account-binding"
	tr := &snap.Transport{}
	req := AccountBindingRequest{
		PartnerReferenceNo: "ref-1",
		MerchantID:         "merchant-1",
		SuccessParams: &BindingSuccessParams{
			AccountID:        "acct-1",
			TerminalID:       "term-1",
			TokenRequestorID: "tr-1",
		},
		AdditionalData: json.RawMessage(`{"deviceId":"12345"}`),
		AdditionalInfo: json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if _, err := AccountBinding(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("AccountBinding() error = %v", err)
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
	if got["merchantId"] != "merchant-1" {
		t.Errorf(`wire body["merchantId"] = %v, want "merchant-1"`, got["merchantId"])
	}
	successParams, ok := got["successParams"].(map[string]any)
	if !ok || successParams["accountId"] != "acct-1" || successParams["terminalId"] != "term-1" || successParams["tokenRequestorId"] != "tr-1" {
		t.Errorf(`wire body["successParams"] = %v, want {accountId:acct-1 terminalId:term-1 tokenRequestorId:tr-1}`, got["successParams"])
	}
	additionalInfo, ok := got["additionalInfo"].(map[string]any)
	if !ok || additionalInfo["channel"] != "mobilephone" {
		t.Errorf(`wire body["additionalInfo"] = %v, want {"channel":"mobilephone"}`, got["additionalInfo"])
	}
	additionalData, ok := got["additionalData"].(map[string]any)
	if !ok || additionalData["deviceId"] != "12345" {
		t.Errorf(`wire body["additionalData"] = %v, want {"deviceId":"12345"}`, got["additionalData"])
	}
}

func TestAccountBinding_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4000700","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-account-binding"
	tr := &snap.Transport{}
	_, err := AccountBinding(context.Background(), tr, hb, AccountBindingRequest{MerchantID: "merchant-1"})
	if err == nil {
		t.Fatal("AccountBinding() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("AccountBinding() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestAccountBinding_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2000700","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-account-binding"
	tr := &snap.Transport{}
	resp, err := AccountBinding(context.Background(), tr, hb, AccountBindingRequest{MerchantID: "merchant-1"})
	if err == nil {
		t.Fatalf("AccountBinding() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("AccountBinding() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestAccountBinding_MerchantIDAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2000700","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-account-binding"
	tr := &snap.Transport{}
	if _, err := AccountBinding(context.Background(), tr, hb, AccountBindingRequest{}); err != nil {
		t.Fatalf("AccountBinding() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	merchantID, ok := got["merchantId"]
	if !ok {
		t.Fatal(`wire body missing "merchantId" key; MerchantID lacks omitempty and must always be present, even as ""`)
	}
	if merchantID != "" {
		t.Errorf(`wire body["merchantId"] = %v, want ""`, merchantID)
	}
}

func TestAccountBinding_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"linkId":"link-1"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-account-binding"
	tr := &snap.Transport{}
	resp, err := AccountBinding(context.Background(), tr, hb, AccountBindingRequest{MerchantID: "merchant-1"})
	if err == nil {
		t.Fatalf("AccountBinding() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
