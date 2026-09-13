package transferdebit

import (
	"encoding/json"
	"reflect"
	"testing"

	snap "github.com/koriebruh/go-snap-bi"
)

func TestDirectDebitPaymentNotificationTypes_FieldCounts(t *testing.T) {
	if n := reflect.TypeOf(DirectDebitPaymentNotificationRequest{}).NumField(); n != 12 {
		t.Errorf("DirectDebitPaymentNotificationRequest has %d fields, want 12", n)
	}
	if n := reflect.TypeOf(DirectDebitPaymentNotificationResponse{}).NumField(); n != 3 {
		t.Errorf("DirectDebitPaymentNotificationResponse has %d fields, want 3", n)
	}
}

func TestDirectDebitPaymentNotificationRequest_RoundTrips(t *testing.T) {
	const fixture = `{
   "originalPartnerReferenceNo":"partner-ref-1",
   "originalReferenceNo":"ref-1",
   "originalExternalId":"ext-1",
   "merchantId":"MERCH01",
   "subMerchantId":"SUBMERCH01",
   "amount":{"value":"50000.00","currency":"IDR"},
   "latestTransactionStatus":"00",
   "transactionStatusDesc":"Success",
   "createdTime":"2026-09-12T09:00:00+07:00",
   "finishedTime":"2026-09-12T09:05:00+07:00",
   "externalStoreId":"STORE01",
   "additionalInfo":{"note":"req-value"}
}`
	var got DirectDebitPaymentNotificationRequest
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := DirectDebitPaymentNotificationRequest{
		OriginalPartnerReferenceNo: "partner-ref-1",
		OriginalReferenceNo:        "ref-1",
		OriginalExternalID:         "ext-1",
		MerchantID:                 "MERCH01",
		SubMerchantID:              "SUBMERCH01",
		Amount:                     &snap.Money{Value: "50000.00", Currency: "IDR"},
		LatestTransactionStatus:    "00",
		TransactionStatusDesc:      "Success",
		CreatedTime:                "2026-09-12T09:00:00+07:00",
		FinishedTime:               "2026-09-12T09:05:00+07:00",
		ExternalStoreID:            "STORE01",
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
	if wire["originalReferenceNo"] != "ref-1" || wire["latestTransactionStatus"] != "00" {
		t.Errorf("marshaled request = %v, want originalReferenceNo/latestTransactionStatus present", wire)
	}
}

// TestDirectDebitPaymentNotificationRequest_MandatoryFieldsHaveNoOmitempty
// pins that OriginalReferenceNo and LatestTransactionStatus — the
// fields without omitempty — always serialize, even from a zero-value
// request.
func TestDirectDebitPaymentNotificationRequest_MandatoryFieldsHaveNoOmitempty(t *testing.T) {
	b, err := json.Marshal(DirectDebitPaymentNotificationRequest{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value request: %v", err)
	}
	want := map[string]any{"originalReferenceNo": "", "latestTransactionStatus": ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value request = %v, want %v (every other field Optional and unset)", got, want)
	}
}

func TestDirectDebitPaymentNotificationResponse_RoundTrips(t *testing.T) {
	const fixture = `{
   "responseCode":"2005600",
   "approvalCode":"APPROVAL001",
   "responseMessage":"Request has been processed successfully"
}`
	var got DirectDebitPaymentNotificationResponse
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := DirectDebitPaymentNotificationResponse{
		ResponseCode:    "2005600",
		ResponseMessage: "Request has been processed successfully",
		ApprovalCode:    "APPROVAL001",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("json.Unmarshal() = %+v, want %+v", got, want)
	}
}

// TestDirectDebitPaymentNotificationResponse_ZeroValueOmitsApprovalCode
// pins that a zero-value response marshals to just the two envelope
// fields, since ApprovalCode is the only Optional field.
func TestDirectDebitPaymentNotificationResponse_ZeroValueOmitsApprovalCode(t *testing.T) {
	b, err := json.Marshal(DirectDebitPaymentNotificationResponse{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value response: %v", err)
	}
	want := map[string]any{"responseCode": "", "responseMessage": ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value response = %v, want %v", got, want)
	}
}
