package snap

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestAuthPaymentTypes_FieldCounts(t *testing.T) {
	if n := reflect.TypeOf(AuthPaymentRequest{}).NumField(); n != 10 {
		t.Errorf("AuthPaymentRequest has %d fields, want 10", n)
	}
	if n := reflect.TypeOf(AuthPaymentResponse{}).NumField(); n != 7 {
		t.Errorf("AuthPaymentResponse has %d fields, want 7", n)
	}
}

func TestAuthPaymentRequest_RoundTrips(t *testing.T) {
	const fixture = `{
   "partnerReferenceNo":"partner-ref-1",
   "merchantId":"MERCHANT001",
   "subMerchantId":"SUBMERCHANT001",
   "amount":{"value":"50000.00","currency":"IDR"},
   "feeType":"OUR",
   "mcc":"5411",
   "productCode":"PROD001",
   "title":"Order #123",
   "items":[{"goodsId":"G1","price":"50000.00","category":"food","unit":"pcs","quantity":"1"}],
   "additionalInfo":{"note":"req-value"}
}`
	var got AuthPaymentRequest
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := AuthPaymentRequest{
		PartnerReferenceNo: "partner-ref-1",
		MerchantID:         "MERCHANT001",
		SubMerchantID:      "SUBMERCHANT001",
		Amount:             &Money{Value: "50000.00", Currency: "IDR"},
		FeeType:            "OUR",
		MCC:                "5411",
		ProductCode:        "PROD001",
		Title:              "Order #123",
		Items:              json.RawMessage(`[{"goodsId":"G1","price":"50000.00","category":"food","unit":"pcs","quantity":"1"}]`),
		AdditionalInfo:     json.RawMessage(`{"note":"req-value"}`),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("json.Unmarshal() = %+v, want %+v", got, want)
	}

	b, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var wire map[string]any
	if err := json.Unmarshal(b, &wire); err != nil {
		t.Fatalf("decode marshaled request: %v", err)
	}
	wantWire := map[string]any{
		"partnerReferenceNo": "partner-ref-1",
		"merchantId":         "MERCHANT001",
		"subMerchantId":      "SUBMERCHANT001",
		"amount":             map[string]any{"value": "50000.00", "currency": "IDR"},
		"feeType":            "OUR",
		"mcc":                "5411",
		"productCode":        "PROD001",
		"title":              "Order #123",
		"items":              []any{map[string]any{"goodsId": "G1", "price": "50000.00", "category": "food", "unit": "pcs", "quantity": "1"}},
		"additionalInfo":     map[string]any{"note": "req-value"},
	}
	if !reflect.DeepEqual(wire, wantWire) {
		t.Errorf("marshaled request = %v, want %v", wire, wantWire)
	}
}

// TestAuthPaymentRequest_MandatoryFieldsHaveNoOmitempty pins that
// PartnerReferenceNo, MerchantID, and Title — the fields without
// omitempty — always serialize, even from a zero-value request.
func TestAuthPaymentRequest_MandatoryFieldsHaveNoOmitempty(t *testing.T) {
	b, err := json.Marshal(AuthPaymentRequest{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value request: %v", err)
	}
	want := map[string]any{
		"partnerReferenceNo": "",
		"merchantId":         "",
		"title":              "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value request = %v, want %v (every other field Optional and unset)", got, want)
	}
}

func TestAuthPaymentResponse_RoundTrips(t *testing.T) {
	const fixture = `{
   "responseCode":"2007300",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"REF001",
   "partnerReferenceNo":"partner-ref-1",
   "amount":{"value":"50000.00","currency":"IDR"},
   "paidTime":"2026-09-13T10:00:00+07:00",
   "additionalInfo":{"note":"resp-value"}
}`
	var got AuthPaymentResponse
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := AuthPaymentResponse{
		ResponseCode:       "2007300",
		ResponseMessage:    "Request has been processed successfully",
		ReferenceNo:        "REF001",
		PartnerReferenceNo: "partner-ref-1",
		Amount:             Money{Value: "50000.00", Currency: "IDR"},
		PaidTime:           "2026-09-13T10:00:00+07:00",
		AdditionalInfo:     json.RawMessage(`{"note":"resp-value"}`),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("json.Unmarshal() = %+v, want %+v", got, want)
	}
}

// TestAuthPaymentResponse_MandatoryFieldsHaveNoOmitempty pins that
// ResponseCode, ResponseMessage, Amount (as a struct value, always
// present), and PaidTime always serialize from a zero-value response.
func TestAuthPaymentResponse_MandatoryFieldsHaveNoOmitempty(t *testing.T) {
	b, err := json.Marshal(AuthPaymentResponse{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value response: %v", err)
	}
	want := map[string]any{
		"responseCode":    "",
		"responseMessage": "",
		"amount":          map[string]any{"value": "", "currency": ""},
		"paidTime":        "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value response = %v, want %v", got, want)
	}
}

func TestAuthPayment_MalformedAdditionalInfoIsMarshalError(t *testing.T) {
	req := AuthPaymentRequest{
		PartnerReferenceNo: "partner-ref-1",
		MerchantID:         "MERCHANT001",
		Title:              "Order #123",
		AdditionalInfo:     json.RawMessage(`{not-valid-json`),
	}
	if _, err := json.Marshal(req); err == nil {
		t.Error("json.Marshal() error = nil, want an error for malformed AdditionalInfo")
	}
}
