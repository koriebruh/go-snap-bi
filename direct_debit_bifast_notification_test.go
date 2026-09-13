package snap

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestDirectDebitBIFASTNotificationTypes_FieldCounts(t *testing.T) {
	if n := reflect.TypeOf(DirectDebitBIFASTNotificationRequest{}).NumField(); n != 11 {
		t.Errorf("DirectDebitBIFASTNotificationRequest has %d fields, want 11", n)
	}
	if n := reflect.TypeOf(DirectDebitBIFASTNotificationResponse{}).NumField(); n != 2 {
		t.Errorf("DirectDebitBIFASTNotificationResponse has %d fields, want 2", n)
	}
}

func TestDirectDebitBIFASTNotificationRequest_RoundTrips(t *testing.T) {
	const fixture = `{
   "originalReferenceNo":"ref-1",
   "originalPartnerReferenceNo":"partner-ref-1",
   "originalExternalId":"ext-1",
   "transactionStatus":"00",
   "transactionStatusDesc":"Success",
   "eMandateReffId":"EMANDATE001",
   "sourceAccountNo":"1234567890123456789012345678901234",
   "sourceAccountName":"Jane Doe",
   "amount":{"value":"50000.00","currency":"IDR"},
   "traceNo":"TRACE001",
   "additionalInfo":{"note":"req-value"}
}`
	var got DirectDebitBIFASTNotificationRequest
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := DirectDebitBIFASTNotificationRequest{
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		OriginalExternalID:         "ext-1",
		TransactionStatus:          "00",
		TransactionStatusDesc:      "Success",
		EMandateReffID:             "EMANDATE001",
		SourceAccountNo:            "1234567890123456789012345678901234",
		SourceAccountName:          "Jane Doe",
		Amount:                     &Money{Value: "50000.00", Currency: "IDR"},
		TraceNo:                    "TRACE001",
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
	if wire["originalReferenceNo"] != "ref-1" || wire["transactionStatus"] != "00" {
		t.Errorf("marshaled request = %v, want originalReferenceNo/transactionStatus present", wire)
	}
}

// TestDirectDebitBIFASTNotificationRequest_MandatoryFieldsHaveNoOmitempty
// pins that OriginalReferenceNo, TransactionStatus, EMandateReffID,
// SourceAccountNo, and SourceAccountName — the fields without
// omitempty — always serialize, even from a zero-value request.
func TestDirectDebitBIFASTNotificationRequest_MandatoryFieldsHaveNoOmitempty(t *testing.T) {
	b, err := json.Marshal(DirectDebitBIFASTNotificationRequest{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value request: %v", err)
	}
	want := map[string]any{
		"originalReferenceNo": "",
		"transactionStatus":   "",
		"eMandateReffId":      "",
		"sourceAccountNo":     "",
		"sourceAccountName":   "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value request = %v, want %v (every other field Optional and unset)", got, want)
	}
}

func TestDirectDebitBIFASTNotificationResponse_RoundTrips(t *testing.T) {
	const fixture = `{
   "responseCode":"2007200",
   "responseMessage":"Request has been processed successfully"
}`
	var got DirectDebitBIFASTNotificationResponse
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := DirectDebitBIFASTNotificationResponse{
		ResponseCode:    "2007200",
		ResponseMessage: "Request has been processed successfully",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("json.Unmarshal() = %+v, want %+v", got, want)
	}
}
