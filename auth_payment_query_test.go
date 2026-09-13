package snap

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestAuthPaymentQueryTypes_FieldCounts(t *testing.T) {
	if n := reflect.TypeOf(AuthPaymentQueryRequest{}).NumField(); n != 6 {
		t.Errorf("AuthPaymentQueryRequest has %d fields, want 6", n)
	}
	if n := reflect.TypeOf(AuthPaymentQueryResponse{}).NumField(); n != 9 {
		t.Errorf("AuthPaymentQueryResponse has %d fields, want 9", n)
	}
}

func TestAuthPaymentQueryRequest_RoundTrips(t *testing.T) {
	const fixture = `{
   "originalPartnerReferenceNo":"partner-ref-1",
   "originalReferenceNo":"ref-1",
   "merchantId":"MERCHANT001",
   "subMerchantId":"SUBMERCHANT001",
   "externalStoreId":"STORE001",
   "additionalInfo":{"note":"req-value"}
}`
	var got AuthPaymentQueryRequest
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := AuthPaymentQueryRequest{
		OriginalPartnerReferenceNo: "partner-ref-1",
		OriginalReferenceNo:        "ref-1",
		MerchantID:                 "MERCHANT001",
		SubMerchantID:              "SUBMERCHANT001",
		ExternalStoreID:            "STORE001",
		AdditionalInfo:             json.RawMessage(`{"note":"req-value"}`),
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
		"originalPartnerReferenceNo": "partner-ref-1",
		"originalReferenceNo":        "ref-1",
		"merchantId":                 "MERCHANT001",
		"subMerchantId":              "SUBMERCHANT001",
		"externalStoreId":            "STORE001",
		"additionalInfo":             map[string]any{"note": "req-value"},
	}
	if !reflect.DeepEqual(wire, wantWire) {
		t.Errorf("marshaled request = %v, want %v", wire, wantWire)
	}
}

// TestAuthPaymentQueryRequest_ZeroValueMarshalsEmpty pins that a
// zero-value request has no Mandatory fields — every key is omitted.
func TestAuthPaymentQueryRequest_ZeroValueMarshalsEmpty(t *testing.T) {
	b, err := json.Marshal(AuthPaymentQueryRequest{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value request: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("marshaled zero-value request = %v, want empty object (no Mandatory field)", got)
	}
}

func TestAuthPaymentQueryResponse_RoundTrips(t *testing.T) {
	const fixture = `{
   "responseCode":"2007400",
   "responseMessage":"Request has been processed successfully",
   "originalPartnerReferenceNo":"partner-ref-1",
   "originalReferenceNo":"ref-1",
   "amount":{"value":"50000.00","currency":"IDR"},
   "paidTime":"2026-09-13T10:00:00+07:00",
   "latestTransactionStatus":"00",
   "transactionStatusDesc":"Success",
   "additionalInfo":{"note":"resp-value"}
}`
	var got AuthPaymentQueryResponse
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := AuthPaymentQueryResponse{
		ResponseCode:               "2007400",
		ResponseMessage:            "Request has been processed successfully",
		OriginalPartnerReferenceNo: "partner-ref-1",
		OriginalReferenceNo:        "ref-1",
		Amount:                     Money{Value: "50000.00", Currency: "IDR"},
		PaidTime:                   "2026-09-13T10:00:00+07:00",
		LatestTransactionStatus:    "00",
		TransactionStatusDesc:      "Success",
		AdditionalInfo:             json.RawMessage(`{"note":"resp-value"}`),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("json.Unmarshal() = %+v, want %+v", got, want)
	}
}

// TestAuthPaymentQueryResponse_MandatoryFieldsHaveNoOmitempty pins
// ResponseCode, ResponseMessage, Amount, PaidTime, and
// LatestTransactionStatus as always-serializing.
func TestAuthPaymentQueryResponse_MandatoryFieldsHaveNoOmitempty(t *testing.T) {
	b, err := json.Marshal(AuthPaymentQueryResponse{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value response: %v", err)
	}
	want := map[string]any{
		"responseCode":            "",
		"responseMessage":         "",
		"amount":                  map[string]any{"value": "", "currency": ""},
		"paidTime":                "",
		"latestTransactionStatus": "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value response = %v, want %v", got, want)
	}
}

// TestAuthPaymentQueryResponse_LowercasePCasingUnmarshalsCorrectly
// pins the resolution of research §6 item 2: the worked example's
// originalpartnerReferenceNo (lowercase p) casing still populates
// OriginalPartnerReferenceNo via Go's case-insensitive JSON decode.
func TestAuthPaymentQueryResponse_LowercasePCasingUnmarshalsCorrectly(t *testing.T) {
	const fixture = `{
   "responseCode":"2007400",
   "responseMessage":"Request has been processed successfully",
   "originalpartnerReferenceNo":"partner-ref-1",
   "amount":{"value":"50000.00","currency":"IDR"},
   "paidTime":"2026-09-13T10:00:00+07:00",
   "latestTransactionStatus":"00"
}`
	var got AuthPaymentQueryResponse
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.OriginalPartnerReferenceNo != "partner-ref-1" {
		t.Errorf("OriginalPartnerReferenceNo = %q, want %q (lowercase-p wire casing must still decode)", got.OriginalPartnerReferenceNo, "partner-ref-1")
	}
}

func TestAuthPaymentQuery_MalformedAdditionalInfoIsMarshalError(t *testing.T) {
	req := AuthPaymentQueryRequest{
		AdditionalInfo: json.RawMessage(`{not-valid-json`),
	}
	if _, err := json.Marshal(req); err == nil {
		t.Error("json.Marshal() error = nil, want an error for malformed AdditionalInfo")
	}
}
