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

func TestInquiryVA_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2003000",
   "responseMessage":"Request has been processed successfully",
   "virtualAccountData":{
      "partnerServiceId":"12345",
      "customerNo":"98765",
      "virtualAccountNo":"1234598765",
      "virtualAccountName":"Jane Doe",
      "trxId":"trx-1",
      "totalAmount":{"value":"100000.00","currency":"IDR"},
      "billDetails":[{
         "billCode":"01",
         "billNo":"bill-1",
         "billName":"Monthly Fee",
         "billShortName":"Fee",
         "billDescription":{"english":"Monthly fee","indonesia":"Biaya bulanan"},
         "billSubCompany":"00001",
         "billAmount":{"value":"50000.00","currency":"IDR"},
         "additionalInfo":{"note":"bill-level"},
         "billAmountLabel":"Total",
         "billAmountValue":"50000.00",
         "billReferenceNo":"BILLREF1",
         "status":"01",
         "reason":{"english":"unpaid","indonesia":"belum dibayar"}
      }],
      "freeTexts":[{"english":"note","indonesia":"catatan"}],
      "virtualAccountTrxType":"C",
      "feeAmount":{"value":"1000.00","currency":"IDR"},
      "expiredDate":"2020-12-21T14:56:11+07:00",
      "lastUpdateDate":"2020-12-22T09:00:00+07:00",
      "paymentDate":"2020-12-22T10:00:00+07:00",
      "additionalInfo":{"channel":"mobilephone"}
   }
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry-va"
	tr := &snap.Transport{}
	resp, err := InquiryVA(context.Background(), tr, hb, InquiryVARequest{
		PartnerServiceID: "12345",
		CustomerNo:       "98765",
		VirtualAccountNo: "1234598765",
		TrxID:            "trx-1",
	})
	if err != nil {
		t.Fatalf("InquiryVA() error = %v", err)
	}

	want := InquiryVAResponse{
		ResponseCode:    "2003000",
		ResponseMessage: "Request has been processed successfully",
		VirtualAccountData: &InquiryVAData{
			PartnerServiceID:   "12345",
			CustomerNo:         "98765",
			VirtualAccountNo:   "1234598765",
			VirtualAccountName: "Jane Doe",
			TrxID:              "trx-1",
			TotalAmount:        &snap.Money{Value: "100000.00", Currency: "IDR"},
			BillDetails: []BillDetail{{
				BillCode:        "01",
				BillNo:          "bill-1",
				BillName:        "Monthly Fee",
				BillShortName:   "Fee",
				BillDescription: &LocalizedText{English: "Monthly fee", Indonesia: "Biaya bulanan"},
				BillSubCompany:  "00001",
				BillAmount:      &snap.Money{Value: "50000.00", Currency: "IDR"},
				AdditionalInfo:  json.RawMessage(`{"note":"bill-level"}`),
				BillAmountLabel: "Total",
				BillAmountValue: "50000.00",
				BillReferenceNo: json.RawMessage(`"BILLREF1"`),
				Status:          "01",
				Reason:          &LocalizedText{English: "unpaid", Indonesia: "belum dibayar"},
			}},
			FreeTexts:             []LocalizedText{{English: "note", Indonesia: "catatan"}},
			VirtualAccountTrxType: "C",
			FeeAmount:             &snap.Money{Value: "1000.00", Currency: "IDR"},
			ExpiredDate:           "2020-12-21T14:56:11+07:00",
			LastUpdateDate:        "2020-12-22T09:00:00+07:00",
			PaymentDate:           "2020-12-22T10:00:00+07:00",
			AdditionalInfo:        json.RawMessage(`{"channel":"mobilephone"}`),
		},
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("InquiryVA() = %+v, want %+v", resp, want)
	}
}

// TestInquiryVA_BillReferenceNoAcceptsEitherWireShape is the regression
// test for a santa-loop finding: the Guides tab labels billReferenceNo
// "Number", and while the worked example renders it quoted, SNAP is a
// multi-PJP standard and one example is thin evidence about what every
// issuer sends. A plain string field would hard-fail the entire decode
// (discarding the VA identity, all other bills, etc.) for an unquoted
// numeric value. json.RawMessage tolerates either shape — same fix,
// same reasoning, as AccountBindingInquiryResponse.AccountTransactionLimit.
func TestInquiryVA_BillReferenceNoAcceptsEitherWireShape(t *testing.T) {
	tests := []struct {
		name string
		ref  string
	}{
		{"quoted string", `"BILLREF1"`},
		{"unquoted number", `123456789012345`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"responseCode":"2003000","responseMessage":"ok","virtualAccountData":{"virtualAccountNo":"1234598765","billDetails":[{"billCode":"01","billReferenceNo":`+tt.ref+`}]}}`)
			}))
			defer server.Close()

			hb := snaptest.TestHeaderBuilder(server.URL)
			hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry-va"
			tr := &snap.Transport{}
			resp, err := InquiryVA(context.Background(), tr, hb, InquiryVARequest{})
			if err != nil {
				t.Fatalf("InquiryVA() error = %v, want nil — billReferenceNo's shape must not break the whole decode", err)
			}
			if resp.VirtualAccountData == nil || resp.VirtualAccountData.VirtualAccountNo != "1234598765" {
				t.Errorf("VirtualAccountNo did not survive regardless of billReferenceNo's shape: %+v", resp.VirtualAccountData)
			}
			if len(resp.VirtualAccountData.BillDetails) != 1 || string(resp.VirtualAccountData.BillDetails[0].BillReferenceNo) != tt.ref {
				t.Errorf("BillReferenceNo = %s, want raw bytes %s", resp.VirtualAccountData.BillDetails[0].BillReferenceNo, tt.ref)
			}
		})
	}
}

// TestInquiryVA_BareNumberResponseCodeIsAKnownLimitation pins a known,
// recorded limitation: research §5.3 shows Inquiry VA's own worked
// example rendering responseCode as a bare JSON number
// (`"responseCode":2003000,`), unlike every other endpoint's worked
// example in the researched Transfer Kredit group. responseCode is
// decoded twice — once by the shared transport layer, and again by
// InquiryVAResponse's own decode — and both are typed string like
// every Response.ResponseCode in the package, so a real server sending
// this shape fails the call at the transport layer first, and would
// fail again at InquiryVAResponse's own decode even if the transport
// layer alone were fixed. See the Phase 13 design doc's "Known
// limitation" section. This test documents the current (failing)
// behavior; it stays meaningfully failing-shaped until both decode
// sites (and, package-wide, every other Response.ResponseCode) are
// retyped — not just the transport layer.
func TestInquiryVA_BareNumberResponseCodeIsAKnownLimitation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"responseCode":2003000,"responseMessage":"ok"}`)
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry-va"
	tr := &snap.Transport{}
	_, err := InquiryVA(context.Background(), tr, hb, InquiryVARequest{})
	if err == nil {
		t.Fatal("InquiryVA() error = nil, want non-nil — this pins a known limitation (bare-number responseCode breaks both the shared transport decode and InquiryVAResponse's own decode); if this now passes, both decode sites have been fixed and this test should be updated to assert success")
	}
}

func TestInquiryVA_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2003000","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry-va"
	tr := &snap.Transport{}
	req := InquiryVARequest{
		PartnerServiceID: "12345",
		CustomerNo:       "98765",
		VirtualAccountNo: "1234598765",
		TrxID:            "trx-1",
		AdditionalInfo:   json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if _, err := InquiryVA(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("InquiryVA() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["partnerServiceId"] != "12345" || got["customerNo"] != "98765" || got["virtualAccountNo"] != "1234598765" || got["trxId"] != "trx-1" {
		t.Errorf("wire body = %v, want the test's identity triple + trxId", got)
	}
}

// TestInquiryVA_MandatoryFieldsAlwaysSerialized pins that
// PartnerServiceID, CustomerNo, VirtualAccountNo, and TrxID — the four
// request fields without omitempty — are always present on the wire,
// even as "".
func TestInquiryVA_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2003000","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry-va"
	tr := &snap.Transport{}
	if _, err := InquiryVA(context.Background(), tr, hb, InquiryVARequest{}); err != nil {
		t.Fatalf("InquiryVA() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	for _, key := range []string{"partnerServiceId", "customerNo", "virtualAccountNo", "trxId"} {
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

func TestInquiryVA_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4003000","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry-va"
	tr := &snap.Transport{}
	_, err := InquiryVA(context.Background(), tr, hb, InquiryVARequest{})
	if err == nil {
		t.Fatal("InquiryVA() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("InquiryVA() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestInquiryVA_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2003000","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry-va"
	tr := &snap.Transport{}
	resp, err := InquiryVA(context.Background(), tr, hb, InquiryVARequest{})
	if err == nil {
		t.Fatalf("InquiryVA() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("InquiryVA() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestInquiryVA_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"virtualAccountData":{"trxId":"trx-1"}}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/transfer-va/inquiry-va"
	tr := &snap.Transport{}
	resp, err := InquiryVA(context.Background(), tr, hb, InquiryVARequest{})
	if err == nil {
		t.Fatalf("InquiryVA() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
