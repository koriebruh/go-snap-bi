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

func TestUpdateStatusVA_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2002900",
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
      "expiredDate":"2020-12-20T00:00:00+07:00",
      "lastUpdateDate":"2020-12-21T14:56:11+07:00",
      "paymentDate":"2020-12-22T09:00:00+07:00",
      "additionalInfo":{"channel":"mobilephone"}
   }
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/update-status"
	tr := &snap.Transport{}
	resp, err := UpdateStatusVA(context.Background(), tr, hb, UpdateStatusVARequest{
		PartnerServiceID: "12345",
		CustomerNo:       "98765",
		VirtualAccountNo: "1234598765",
		TrxID:            "trx-1",
		PaidStatus:       "Y",
	})
	if err != nil {
		t.Fatalf("UpdateStatusVA() error = %v", err)
	}

	want := UpdateStatusVAResponse{
		ResponseCode:    "2002900",
		ResponseMessage: "Request has been processed successfully",
		VirtualAccountData: &UpdateStatusVAData{
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
			ExpiredDate:           "2020-12-20T00:00:00+07:00",
			LastUpdateDate:        "2020-12-21T14:56:11+07:00",
			PaymentDate:           "2020-12-22T09:00:00+07:00",
			AdditionalInfo:        json.RawMessage(`{"channel":"mobilephone"}`),
		},
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("UpdateStatusVA() = %+v, want %+v", resp, want)
	}
}

// TestUpdateStatusVA_UsesPUTMethod pins that UpdateStatusVA sets
// hb.Method to PUT regardless of what the caller configured.
func TestUpdateStatusVA_UsesPUTMethod(t *testing.T) {
	var mu sync.Mutex
	var gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotMethod = r.Method
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"responseCode":"2002900","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/update-status"
	tr := &snap.Transport{}
	if _, err := UpdateStatusVA(context.Background(), tr, hb, UpdateStatusVARequest{}); err != nil {
		t.Fatalf("UpdateStatusVA() error = %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if gotMethod != http.MethodPut {
		t.Errorf("request method = %q, want %q", gotMethod, http.MethodPut)
	}
}

func TestUpdateStatusVA_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2002900","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/update-status"
	tr := &snap.Transport{}
	req := UpdateStatusVARequest{
		PartnerServiceID: "12345",
		CustomerNo:       "98765",
		VirtualAccountNo: "1234598765",
		TrxID:            "trx-1",
		PaidStatus:       "Y",
		AdditionalInfo:   json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if _, err := UpdateStatusVA(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("UpdateStatusVA() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["paidStatus"] != "Y" {
		t.Errorf(`wire body["paidStatus"] = %v, want "Y"`, got["paidStatus"])
	}
	if got["trxId"] != "trx-1" {
		t.Errorf(`wire body["trxId"] = %v, want "trx-1"`, got["trxId"])
	}
}

// TestUpdateStatusVA_MandatoryFieldsAlwaysSerialized pins that
// PartnerServiceID, CustomerNo, VirtualAccountNo, TrxID, and
// PaidStatus — the five request fields without omitempty — are always
// present on the wire, even as "".
func TestUpdateStatusVA_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2002900","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/update-status"
	tr := &snap.Transport{}
	if _, err := UpdateStatusVA(context.Background(), tr, hb, UpdateStatusVARequest{}); err != nil {
		t.Fatalf("UpdateStatusVA() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	for _, key := range []string{"partnerServiceId", "customerNo", "virtualAccountNo", "trxId", "paidStatus"} {
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

func TestUpdateStatusVA_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4002900","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/update-status"
	tr := &snap.Transport{}
	_, err := UpdateStatusVA(context.Background(), tr, hb, UpdateStatusVARequest{})
	if err == nil {
		t.Fatal("UpdateStatusVA() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("UpdateStatusVA() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestUpdateStatusVA_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2002900","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/update-status"
	tr := &snap.Transport{}
	resp, err := UpdateStatusVA(context.Background(), tr, hb, UpdateStatusVARequest{})
	if err == nil {
		t.Fatalf("UpdateStatusVA() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("UpdateStatusVA() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestUpdateStatusVA_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"virtualAccountData":{"trxId":"trx-1"}}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/update-status"
	tr := &snap.Transport{}
	resp, err := UpdateStatusVA(context.Background(), tr, hb, UpdateStatusVARequest{})
	if err == nil {
		t.Fatalf("UpdateStatusVA() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
