package transferkredit

import (
	"encoding/json"
	"reflect"
	"testing"

	snap "github.com/koriebruh/go-snap-bi"
)

func TestRTGSNotificationRequest_RoundTrips(t *testing.T) {
	const fixture = `{
   "originalPartnerReferenceNo":"2020102900000000000001",
   "originalReferenceNo":"2020102977770000000009",
   "originalExternalId":"ext-1",
   "latestTransactionStatus":"00",
   "amount":{"value":"50000.00","currency":"IDR"},
   "beneficiaryAccountName":"Jane Doe",
   "beneficiaryAccountNo":"1234567890",
   "beneficiaryBankCode":"014",
   "sourceAccountNo":"9876543210",
   "transactionDate":"2020-12-21T14:56:11+07:00",
   "additionalInfo":{"channel":"mobilephone"}
}`
	var got RTGSNotificationRequest
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := RTGSNotificationRequest{
		OriginalPartnerReferenceNo: "2020102900000000000001",
		OriginalReferenceNo:        "2020102977770000000009",
		OriginalExternalID:         "ext-1",
		LatestTransactionStatus:    "00",
		Amount:                     &snap.Money{Value: "50000.00", Currency: "IDR"},
		BeneficiaryAccountName:     "Jane Doe",
		BeneficiaryAccountNo:       "1234567890",
		BeneficiaryBankCode:        "014",
		SourceAccountNo:            "9876543210",
		TransactionDate:            "2020-12-21T14:56:11+07:00",
		AdditionalInfo:             json.RawMessage(`{"channel":"mobilephone"}`),
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
	if wire["latestTransactionStatus"] != "00" || wire["beneficiaryAccountName"] != "Jane Doe" ||
		wire["beneficiaryAccountNo"] != "1234567890" || wire["beneficiaryBankCode"] != "014" ||
		wire["sourceAccountNo"] != "9876543210" || wire["transactionDate"] != "2020-12-21T14:56:11+07:00" {
		t.Errorf("marshaled request = %v, want every mandatory field present", wire)
	}

	// A zero-value marshal is the actual omitempty guard: a fully-populated
	// marshal above would still pass even if a mandatory field regressed to
	// carrying omitempty, since no field would be empty either way.
	zb, err := json.Marshal(RTGSNotificationRequest{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var zeroWire map[string]any
	if err := json.Unmarshal(zb, &zeroWire); err != nil {
		t.Fatalf("decode marshaled zero-value request: %v", err)
	}
	for _, key := range []string{"latestTransactionStatus", "beneficiaryAccountName", "beneficiaryAccountNo", "beneficiaryBankCode", "sourceAccountNo", "transactionDate"} {
		v, ok := zeroWire[key]
		if !ok {
			t.Errorf(`zero-value marshaled request missing %q key; want it always present, even as ""`, key)
			continue
		}
		if v != "" {
			t.Errorf(`zero-value marshaled request[%q] = %v, want ""`, key, v)
		}
	}
}

func TestRTGSNotificationResponse_RoundTrips(t *testing.T) {
	const fixture = `{"responseCode":"2007600","responseMessage":"Request has been processed successfully"}`
	var got RTGSNotificationResponse
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := RTGSNotificationResponse{ResponseCode: "2007600", ResponseMessage: "Request has been processed successfully"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("json.Unmarshal() = %+v, want %+v", got, want)
	}

	b, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var wire map[string]any
	if err := json.Unmarshal(b, &wire); err != nil {
		t.Fatalf("decode marshaled response: %v", err)
	}
	if wire["responseCode"] != "2007600" || wire["responseMessage"] != "Request has been processed successfully" {
		t.Errorf("marshaled response = %v, want both fields present", wire)
	}
	if len(wire) != 2 {
		t.Errorf("marshaled response = %v, want exactly 2 keys (envelope-only)", wire)
	}
}
