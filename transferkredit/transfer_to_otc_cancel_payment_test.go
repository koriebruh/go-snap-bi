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

func TestTransferToOTCCancelPayment_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2004600",
   "responseMessage":"Request has been processed successfully",
   "originalReferenceNo":"ref-1",
   "cancelTime":"2020-12-20T10:00:00+07:00",
   "transactionDate":"2020-12-20T09:00:00+07:00"
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/otc-cancel"
	tr := &snap.Transport{}
	resp, err := TransferToOTCCancelPayment(context.Background(), tr, hb, TransferToOTCCancelPaymentRequest{
		OriginalPartnerReferenceNo: "partner-ref-1",
		CustomerNumber:             "98765",
		Reason:                     "customer requested cancellation",
	})
	if err != nil {
		t.Fatalf("TransferToOTCCancelPayment() error = %v", err)
	}

	want := TransferToOTCCancelPaymentResponse{
		ResponseCode:        "2004600",
		ResponseMessage:     "Request has been processed successfully",
		OriginalReferenceNo: "ref-1",
		CancelTime:          "2020-12-20T10:00:00+07:00",
		TransactionDate:     "2020-12-20T09:00:00+07:00",
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("TransferToOTCCancelPayment() = %+v, want %+v", resp, want)
	}
}

func TestTransferToOTCCancelPayment_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2004600","responseMessage":"ok","originalReferenceNo":"ref-1"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/otc-cancel"
	tr := &snap.Transport{}
	req := TransferToOTCCancelPaymentRequest{
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		CustomerNumber:             "98765",
		Reason:                     "customer requested cancellation",
	}
	if _, err := TransferToOTCCancelPayment(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("TransferToOTCCancelPayment() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["reason"] != "customer requested cancellation" {
		t.Errorf(`wire body["reason"] = %v, want "customer requested cancellation"`, got["reason"])
	}
	if got["originalReferenceNo"] != "ref-1" {
		t.Errorf(`wire body["originalReferenceNo"] = %v, want "ref-1"`, got["originalReferenceNo"])
	}
}

// TestTransferToOTCCancelPayment_MandatoryFieldsAlwaysSerialized pins
// that OriginalPartnerReferenceNo, CustomerNumber, and Reason — the
// request fields without omitempty — are always present on the wire,
// and that OriginalReferenceNo (Conditional here) is omitted when
// unset.
func TestTransferToOTCCancelPayment_MandatoryFieldsAlwaysSerialized(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2004600","responseMessage":"ok","originalReferenceNo":"ref-1"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/otc-cancel"
	tr := &snap.Transport{}
	if _, err := TransferToOTCCancelPayment(context.Background(), tr, hb, TransferToOTCCancelPaymentRequest{}); err != nil {
		t.Fatalf("TransferToOTCCancelPayment() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	for _, key := range []string{"originalPartnerReferenceNo", "customerNumber", "reason"} {
		v, ok := got[key]
		if !ok {
			t.Errorf(`wire body missing %q key; want it always present, even as ""`, key)
			continue
		}
		if v != "" {
			t.Errorf(`wire body[%q] = %v, want ""`, key, v)
		}
	}
	if _, ok := got["originalReferenceNo"]; ok {
		t.Error(`wire body has "originalReferenceNo" key, want it omitted (Conditional here)`)
	}
}

// TestTransferToOTCCancelPaymentResponse_OriginalReferenceNoHasNoOmitempty
// pins that OriginalReferenceNo — Mandatory in the response, unlike
// Conditional in the request, per research §5.8's explicit note —
// carries no omitempty tag. Decoding an absent JSON key into a Go
// string always yields "" regardless of the struct tag, so the only
// way to actually observe whether omitempty is present is on the
// marshal side: a zero-value marshal always emits the key here, which
// would not be true if omitempty were (re)added by a future edit.
func TestTransferToOTCCancelPaymentResponse_OriginalReferenceNoHasNoOmitempty(t *testing.T) {
	b, err := json.Marshal(TransferToOTCCancelPaymentResponse{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value response: %v", err)
	}
	v, ok := got["originalReferenceNo"]
	if !ok {
		t.Fatal(`marshaled zero-value response missing "originalReferenceNo" key; OriginalReferenceNo lacks omitempty and must always be present`)
	}
	if v != "" {
		t.Errorf(`marshaled zero-value response["originalReferenceNo"] = %v, want ""`, v)
	}
}

func TestTransferToOTCCancelPayment_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4004600","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/otc-cancel"
	tr := &snap.Transport{}
	_, err := TransferToOTCCancelPayment(context.Background(), tr, hb, TransferToOTCCancelPaymentRequest{})
	if err == nil {
		t.Fatal("TransferToOTCCancelPayment() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("TransferToOTCCancelPayment() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestTransferToOTCCancelPayment_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2004600","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/otc-cancel"
	tr := &snap.Transport{}
	resp, err := TransferToOTCCancelPayment(context.Background(), tr, hb, TransferToOTCCancelPaymentRequest{})
	if err == nil {
		t.Fatalf("TransferToOTCCancelPayment() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("TransferToOTCCancelPayment() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestTransferToOTCCancelPayment_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"originalReferenceNo":"ref-1"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/emoney/otc-cancel"
	tr := &snap.Transport{}
	resp, err := TransferToOTCCancelPayment(context.Background(), tr, hb, TransferToOTCCancelPaymentRequest{})
	if err == nil {
		t.Fatalf("TransferToOTCCancelPayment() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
