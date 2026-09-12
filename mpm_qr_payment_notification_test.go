package snap

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestQRMPMPaymentNotificationRequest_RoundTrips(t *testing.T) {
	const fixture = `{
   "originalReferenceNo":"ref-1",
   "originalPartnerReferenceNo":"partner-ref-1",
   "latestTransactionStatus":"00",
   "customerNumber":"98765",
   "accountType":"01",
   "destinationNumber":"1122334455",
   "destinationAccountName":"Jane Doe",
   "amount":{"value":"25000.00","currency":"IDR"},
   "sessionId":"session-1",
   "bankCode":"014",
   "externalStoreId":"STORE01"
}`
	var got QRMPMPaymentNotificationRequest
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := QRMPMPaymentNotificationRequest{
		OriginalReferenceNo:        "ref-1",
		OriginalPartnerReferenceNo: "partner-ref-1",
		LatestTransactionStatus:    "00",
		CustomerNumber:             "98765",
		AccountType:                "01",
		DestinationNumber:          "1122334455",
		DestinationAccountName:     "Jane Doe",
		Amount:                     &Money{Value: "25000.00", Currency: "IDR"},
		SessionID:                  "session-1",
		BankCode:                   "014",
		ExternalStoreID:            "STORE01",
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
	amount, ok := wire["amount"].(map[string]any)
	if !ok || amount["value"] != "25000.00" {
		t.Errorf(`marshaled request["amount"] = %v, want {"value":"25000.00","currency":"IDR"}`, wire["amount"])
	}
}

// TestQRMPMPaymentNotificationRequest_MandatoryFieldsHaveNoOmitempty
// pins that OriginalReferenceNo and LatestTransactionStatus — the
// fields without omitempty — always serialize, even from a zero-value
// request.
func TestQRMPMPaymentNotificationRequest_MandatoryFieldsHaveNoOmitempty(t *testing.T) {
	b, err := json.Marshal(QRMPMPaymentNotificationRequest{})
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

func TestQRMPMPaymentNotificationResponse_RoundTrips(t *testing.T) {
	const fixture = `{
   "responseCode":"2005200",
   "responseMessage":"Request has been processed successfully"
}`
	var got QRMPMPaymentNotificationResponse
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := QRMPMPaymentNotificationResponse{
		ResponseCode:    "2005200",
		ResponseMessage: "Request has been processed successfully",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("json.Unmarshal() = %+v, want %+v", got, want)
	}
}

// TestQRMPMPaymentNotificationResponse_IsEnvelopeOnly pins that the
// response has exactly the two envelope fields — research documents it
// as "envelope-only" (§5.9 line 202); a future edit adding a field
// without updating this test would be a signal to re-check that claim.
func TestQRMPMPaymentNotificationResponse_IsEnvelopeOnly(t *testing.T) {
	typ := reflect.TypeOf(QRMPMPaymentNotificationResponse{})
	if typ.NumField() != 2 {
		names := make([]string, typ.NumField())
		for i := range names {
			names[i] = typ.Field(i).Name
		}
		t.Errorf("QRMPMPaymentNotificationResponse fields = %v, want exactly [ResponseCode ResponseMessage]", names)
	}
}
