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

func TestAccountUnbinding_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2000900",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"2020102977770000000009",
   "partnerReferenceNo":"2020102900000000000001",
   "merchantId":"00007100010926",
   "subMerchantId":"310928924949487",
   "linkId":"abcd1234efgh5678ijkl9012",
   "unlinkResult":"success",
   "additionalInfo":{"channel":"mobilephone"}
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-account-unbinding"
	tr := &snap.Transport{}
	resp, err := AccountUnbinding(context.Background(), tr, hb, AccountUnbindingRequest{MerchantID: "00007100010926"})
	if err != nil {
		t.Fatalf("AccountUnbinding() error = %v", err)
	}

	want := AccountUnbindingResponse{
		ResponseCode:       "2000900",
		ResponseMessage:    "Request has been processed successfully",
		ReferenceNo:        "2020102977770000000009",
		PartnerReferenceNo: "2020102900000000000001",
		MerchantID:         "00007100010926",
		SubMerchantID:      "310928924949487",
		LinkID:             "abcd1234efgh5678ijkl9012",
		UnlinkResult:       "success",
		AdditionalInfo:     json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("AccountUnbinding() = %+v, want %+v", resp, want)
	}
}

func TestAccountUnbinding_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2000900","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-account-unbinding"
	tr := &snap.Transport{}
	req := AccountUnbindingRequest{
		PartnerReferenceNo: "ref-1",
		LinkID:             "abcd1234efgh5678ijkl9012",
		MerchantID:         "00007100010926",
		SubMerchantID:      "310928924949487",
		TokenID:            "Aeox320xvijwefop10",
		AdditionalInfo:     json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if _, err := AccountUnbinding(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("AccountUnbinding() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["merchantId"] != "00007100010926" {
		t.Errorf(`wire body["merchantId"] = %v, want "00007100010926"`, got["merchantId"])
	}
	if got["linkId"] != "abcd1234efgh5678ijkl9012" {
		t.Errorf(`wire body["linkId"] = %v, want "abcd1234efgh5678ijkl9012"`, got["linkId"])
	}
	if got["tokenId"] != "Aeox320xvijwefop10" {
		t.Errorf(`wire body["tokenId"] = %v, want "Aeox320xvijwefop10"`, got["tokenId"])
	}
	additionalInfo, ok := got["additionalInfo"].(map[string]any)
	if !ok || additionalInfo["channel"] != "mobilephone" {
		t.Errorf(`wire body["additionalInfo"] = %v, want {"channel":"mobilephone"}`, got["additionalInfo"])
	}
}

func TestAccountUnbinding_MerchantIDAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2000900","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-account-unbinding"
	tr := &snap.Transport{}
	if _, err := AccountUnbinding(context.Background(), tr, hb, AccountUnbindingRequest{}); err != nil {
		t.Fatalf("AccountUnbinding() error = %v", err)
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

func TestAccountUnbinding_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4000900","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-account-unbinding"
	tr := &snap.Transport{}
	_, err := AccountUnbinding(context.Background(), tr, hb, AccountUnbindingRequest{MerchantID: "00007100010926"})
	if err == nil {
		t.Fatal("AccountUnbinding() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("AccountUnbinding() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestAccountUnbinding_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2000900","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-account-unbinding"
	tr := &snap.Transport{}
	resp, err := AccountUnbinding(context.Background(), tr, hb, AccountUnbindingRequest{MerchantID: "00007100010926"})
	if err == nil {
		t.Fatalf("AccountUnbinding() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("AccountUnbinding() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestAccountUnbinding_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"unlinkResult":"success"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/registration-account-unbinding"
	tr := &snap.Transport{}
	resp, err := AccountUnbinding(context.Background(), tr, hb, AccountUnbindingRequest{MerchantID: "00007100010926"})
	if err == nil {
		t.Fatalf("AccountUnbinding() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
