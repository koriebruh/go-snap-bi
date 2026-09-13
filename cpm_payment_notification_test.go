package snap

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestCPMPaymentNotificationTypes_FieldCounts(t *testing.T) {
	if n := reflect.TypeOf(CPMPaymentNotificationRequest{}).NumField(); n != 15 {
		t.Errorf("CPMPaymentNotificationRequest has %d fields, want 15", n)
	}
	if n := reflect.TypeOf(CPMPaymentNotificationResponse{}).NumField(); n != 2 {
		t.Errorf("CPMPaymentNotificationResponse has %d fields, want 2", n)
	}
}

func TestCPMPaymentNotificationRequest_RoundTrips(t *testing.T) {
	const fixture = `{
   "originalPartnerReferenceNo":"partner-ref-1",
   "originalReferenceNo":"ref-1",
   "merchantId":"MERCH01",
   "subMerchantId":"SUBMERCH01",
   "externalStoreId":"STORE01",
   "amount":{"value":"50000.00","currency":"IDR"},
   "latestTransactionStatus":"00",
   "transactionStatusDesc":"Success",
   "customerNumber":"98765",
   "accountType":"01",
   "destinationNumber":"1122334455",
   "destinationAccountName":"Jane Doe",
   "sessionId":"session-1",
   "bankCode":"014",
   "additionalInfo":{"note":"req-value"}
}`
	var got CPMPaymentNotificationRequest
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := CPMPaymentNotificationRequest{
		OriginalPartnerReferenceNo: "partner-ref-1",
		OriginalReferenceNo:        "ref-1",
		MerchantID:                 "MERCH01",
		SubMerchantID:              "SUBMERCH01",
		ExternalStoreID:            "STORE01",
		Amount:                     &Money{Value: "50000.00", Currency: "IDR"},
		LatestTransactionStatus:    "00",
		TransactionStatusDesc:      "Success",
		CustomerNumber:             "98765",
		AccountType:                "01",
		DestinationNumber:          "1122334455",
		DestinationAccountName:     "Jane Doe",
		SessionID:                  "session-1",
		BankCode:                   "014",
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
	if wire["merchantId"] != "MERCH01" || wire["latestTransactionStatus"] != "00" {
		t.Errorf("marshaled request = %v, want merchantId/latestTransactionStatus present", wire)
	}
}

// TestCPMPaymentNotificationRequest_MandatoryFieldsHaveNoOmitempty
// pins that MerchantID and LatestTransactionStatus — the fields
// without omitempty — always serialize, even from a zero-value
// request.
func TestCPMPaymentNotificationRequest_MandatoryFieldsHaveNoOmitempty(t *testing.T) {
	b, err := json.Marshal(CPMPaymentNotificationRequest{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value request: %v", err)
	}
	want := map[string]any{"merchantId": "", "latestTransactionStatus": ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value request = %v, want %v (every other field Optional and unset)", got, want)
	}
}

func TestCPMPaymentNotificationResponse_RoundTrips(t *testing.T) {
	const fixture = `{
   "responseCode":"2007900",
   "responseMessage":"Request has been processed successfully"
}`
	var got CPMPaymentNotificationResponse
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := CPMPaymentNotificationResponse{
		ResponseCode:    "2007900",
		ResponseMessage: "Request has been processed successfully",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("json.Unmarshal() = %+v, want %+v", got, want)
	}
}

// TestCPMPaymentNotificationResponse_IsEnvelopeOnly pins the field
// count only — research documents this response as "envelope-only"; a
// future edit adding a field without updating this test would be a
// signal to re-check that claim.
func TestCPMPaymentNotificationResponse_IsEnvelopeOnly(t *testing.T) {
	typ := reflect.TypeOf(CPMPaymentNotificationResponse{})
	if typ.NumField() != 2 {
		names := make([]string, typ.NumField())
		for i := range names {
			names[i] = typ.Field(i).Name
		}
		t.Errorf("CPMPaymentNotificationResponse fields = %v, want exactly [ResponseCode ResponseMessage]", names)
	}
}
