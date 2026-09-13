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

func TestCreateVA_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2002700",
   "responseMessage":"Request has been processed successfully",
   "virtualAccountData":{
      "partnerServiceId":"12345",
      "customerNo":"98765",
      "virtualAccountNo":"1234598765",
      "virtualAccountName":"Jane Doe",
      "trxId":"trx-1",
      "totalAmount":{"value":"100000.00","currency":"IDR"},
      "billDetails":[{"billCode":"01","billNo":"bill-1"}],
      "freeTexts":[{"english":"note","indonesia":"catatan"}],
      "virtualAccountTrxType":"C",
      "feeAmount":{"value":"1000.00","currency":"IDR"},
      "expiredDate":"2020-12-21T14:56:11+07:00",
      "additionalInfo":{"channel":"mobilephone"}
   }
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/create-va"
	tr := &snap.Transport{}
	resp, err := CreateVA(context.Background(), tr, hb, CreateVARequest{
		VirtualAccountName: "Jane Doe",
		TrxID:              "trx-1",
	})
	if err != nil {
		t.Fatalf("CreateVA() error = %v", err)
	}

	want := CreateVAResponse{
		ResponseCode:    "2002700",
		ResponseMessage: "Request has been processed successfully",
		VirtualAccountData: &CreateVAData{
			PartnerServiceID:      "12345",
			CustomerNo:            "98765",
			VirtualAccountNo:      "1234598765",
			VirtualAccountName:    "Jane Doe",
			TrxID:                 "trx-1",
			TotalAmount:           &snap.Money{Value: "100000.00", Currency: "IDR"},
			BillDetails:           []BillDetail{{BillCode: "01", BillNo: "bill-1"}},
			FreeTexts:             []LocalizedText{{English: "note", Indonesia: "catatan"}},
			VirtualAccountTrxType: "C",
			FeeAmount:             &snap.Money{Value: "1000.00", Currency: "IDR"},
			ExpiredDate:           "2020-12-21T14:56:11+07:00",
			AdditionalInfo:        json.RawMessage(`{"channel":"mobilephone"}`),
		},
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("CreateVA() = %+v, want %+v", resp, want)
	}
}

func TestCreateVA_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2002700","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/create-va"
	tr := &snap.Transport{}
	req := CreateVARequest{
		PartnerServiceID:   "12345",
		CustomerNo:         "98765",
		VirtualAccountNo:   "1234598765",
		VirtualAccountName: "Jane Doe",
		TrxID:              "trx-1",
		BillDetails: []BillDetail{
			{BillCode: "01", BillNo: "bill-1"},
		},
		FreeTexts: []LocalizedText{
			{English: "note", Indonesia: "catatan"},
		},
		AdditionalInfo: json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if _, err := CreateVA(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("CreateVA() error = %v", err)
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
	if got["virtualAccountName"] != "Jane Doe" || got["trxId"] != "trx-1" {
		t.Errorf("wire body[virtualAccountName/trxId] = %v/%v, want \"Jane Doe\"/\"trx-1\"", got["virtualAccountName"], got["trxId"])
	}
	billDetails, ok := got["billDetails"].([]any)
	if !ok || len(billDetails) != 1 {
		t.Fatalf(`wire body["billDetails"] = %v, want a 1-element array`, got["billDetails"])
	}
	freeTexts, ok := got["freeTexts"].([]any)
	if !ok || len(freeTexts) != 1 {
		t.Fatalf(`wire body["freeTexts"] = %v, want a 1-element array`, got["freeTexts"])
	}
}

// TestCreateVA_MandatoryFieldsAlwaysSerialized pins that
// VirtualAccountName and TrxID — the two request fields without
// omitempty — are always present on the wire, even as "". Unlike every
// other VA endpoint, the identity triple is Optional here and so is not
// asserted present.
func TestCreateVA_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2002700","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/create-va"
	tr := &snap.Transport{}
	if _, err := CreateVA(context.Background(), tr, hb, CreateVARequest{}); err != nil {
		t.Fatalf("CreateVA() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	for _, key := range []string{"virtualAccountName", "trxId"} {
		v, ok := got[key]
		if !ok {
			t.Errorf(`wire body missing %q key; want it always present, even as ""`, key)
			continue
		}
		if v != "" {
			t.Errorf(`wire body[%q] = %v, want ""`, key, v)
		}
	}
	for _, key := range []string{"partnerServiceId", "customerNo", "virtualAccountNo"} {
		if _, ok := got[key]; ok {
			t.Errorf(`wire body has %q key, want it omitted (identity triple is Optional here, unlike other VA endpoints)`, key)
		}
	}
}

func TestCreateVA_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4002700","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/create-va"
	tr := &snap.Transport{}
	_, err := CreateVA(context.Background(), tr, hb, CreateVARequest{})
	if err == nil {
		t.Fatal("CreateVA() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("CreateVA() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestCreateVA_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2002700","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/create-va"
	tr := &snap.Transport{}
	resp, err := CreateVA(context.Background(), tr, hb, CreateVARequest{})
	if err == nil {
		t.Fatalf("CreateVA() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("CreateVA() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestCreateVA_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"virtualAccountData":{"trxId":"trx-1"}}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/create-va"
	tr := &snap.Transport{}
	resp, err := CreateVA(context.Background(), tr, hb, CreateVARequest{})
	if err == nil {
		t.Fatalf("CreateVA() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
