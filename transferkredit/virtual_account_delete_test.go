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

func TestDeleteVA_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2003100",
   "responseMessage":"Request has been processed successfully",
   "virtualAccountData":{
      "partnerServiceId":"12345",
      "customerNo":"98765",
      "virtualAccountNo":"1234598765",
      "trxId":"trx-1",
      "additionalInfo":{"channel":"mobilephone"}
   }
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/delete-va"
	tr := &snap.Transport{}
	resp, err := DeleteVA(context.Background(), tr, hb, DeleteVARequest{
		PartnerServiceID: "12345",
		CustomerNo:       "98765",
		VirtualAccountNo: "1234598765",
	})
	if err != nil {
		t.Fatalf("DeleteVA() error = %v", err)
	}

	want := DeleteVAResponse{
		ResponseCode:    "2003100",
		ResponseMessage: "Request has been processed successfully",
		VirtualAccountData: &DeleteVAData{
			PartnerServiceID: "12345",
			CustomerNo:       "98765",
			VirtualAccountNo: "1234598765",
			TrxID:            "trx-1",
			AdditionalInfo:   json.RawMessage(`{"channel":"mobilephone"}`),
		},
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("DeleteVA() = %+v, want %+v", resp, want)
	}
}

// TestDeleteVA_UsesDELETEMethod pins that DeleteVA sets hb.Method to
// DELETE regardless of what the caller configured, and that the
// request still carries a JSON body (this DELETE ships a body, no path
// parameters, per the research doc).
func TestDeleteVA_UsesDELETEMethod(t *testing.T) {
	var mu sync.Mutex
	var gotMethod string
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		mu.Lock()
		gotMethod = r.Method
		gotBody = b
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"responseCode":"2003100","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL) // snaptest.TestHeaderBuilder sets Method to POST
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/delete-va"
	tr := &snap.Transport{}
	if _, err := DeleteVA(context.Background(), tr, hb, DeleteVARequest{VirtualAccountNo: "1234598765"}); err != nil {
		t.Fatalf("DeleteVA() error = %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if gotMethod != http.MethodDelete {
		t.Errorf("request method = %q, want %q", gotMethod, http.MethodDelete)
	}
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["virtualAccountNo"] != "1234598765" {
		t.Errorf(`DELETE request body["virtualAccountNo"] = %v, want "1234598765" (a body, not path params)`, got["virtualAccountNo"])
	}
}

func TestDeleteVA_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2003100","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/delete-va"
	tr := &snap.Transport{}
	req := DeleteVARequest{
		PartnerServiceID: "12345",
		CustomerNo:       "98765",
		VirtualAccountNo: "1234598765",
		TrxID:            "trx-1",
		AdditionalInfo:   json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if _, err := DeleteVA(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("DeleteVA() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["partnerServiceId"] != "12345" || got["customerNo"] != "98765" || got["virtualAccountNo"] != "1234598765" {
		t.Errorf("wire body identity triple = %v, want the test's values", got)
	}
	if got["trxId"] != "trx-1" {
		t.Errorf(`wire body["trxId"] = %v, want "trx-1"`, got["trxId"])
	}
}

// TestDeleteVA_MandatoryFieldsAlwaysSerialized pins that
// PartnerServiceID, CustomerNo, and VirtualAccountNo — the three
// request fields without omitempty — are always present on the wire,
// even as "". TrxID is Optional here, unlike Inquiry/Update Status VA.
func TestDeleteVA_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2003100","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/delete-va"
	tr := &snap.Transport{}
	if _, err := DeleteVA(context.Background(), tr, hb, DeleteVARequest{}); err != nil {
		t.Fatalf("DeleteVA() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	for _, key := range []string{"partnerServiceId", "customerNo", "virtualAccountNo"} {
		v, ok := got[key]
		if !ok {
			t.Errorf(`wire body missing %q key; want it always present, even as ""`, key)
			continue
		}
		if v != "" {
			t.Errorf(`wire body[%q] = %v, want ""`, key, v)
		}
	}
	if _, ok := got["trxId"]; ok {
		t.Error(`wire body has "trxId" key, want it omitted (Optional here)`)
	}
}

func TestDeleteVA_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4003100","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/delete-va"
	tr := &snap.Transport{}
	_, err := DeleteVA(context.Background(), tr, hb, DeleteVARequest{})
	if err == nil {
		t.Fatal("DeleteVA() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("DeleteVA() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestDeleteVA_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2003100","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/delete-va"
	tr := &snap.Transport{}
	resp, err := DeleteVA(context.Background(), tr, hb, DeleteVARequest{})
	if err == nil {
		t.Fatalf("DeleteVA() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("DeleteVA() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestDeleteVA_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"virtualAccountData":{"trxId":"trx-1"}}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/delete-va"
	tr := &snap.Transport{}
	resp, err := DeleteVA(context.Background(), tr, hb, DeleteVARequest{})
	if err == nil {
		t.Fatalf("DeleteVA() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
