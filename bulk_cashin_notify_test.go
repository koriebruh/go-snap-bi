package snap

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestNotifyBulkCashInRequest_RoundTrips(t *testing.T) {
	const fixture = `{
   "bulkId":"BULK000001",
   "partnerBulkId":"partner-bulk-1",
   "bulkObject":[{"customerNumber":"98765","customerName":"Jane Doe","amount":{"value":"100000.00","currency":"IDR"},"referenceNo":"ref-1","partnerReferenceNo":"partner-ref-1","responseCode":"2004100","responseMessage":"Success","additionalInfo":{"channel":"mobilephone"}}]
}`
	var got NotifyBulkCashInRequest
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := NotifyBulkCashInRequest{
		BulkID:        "BULK000001",
		PartnerBulkID: "partner-bulk-1",
		BulkObject: []BulkCashInNotificationItem{
			{
				CustomerNumber:     "98765",
				CustomerName:       "Jane Doe",
				Amount:             &Money{Value: "100000.00", Currency: "IDR"},
				ReferenceNo:        "ref-1",
				PartnerReferenceNo: "partner-ref-1",
				ResponseCode:       "2004100",
				ResponseMessage:    "Success",
				AdditionalInfo:     json.RawMessage(`{"channel":"mobilephone"}`),
			},
		},
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
	if wire["bulkId"] != "BULK000001" || wire["partnerBulkId"] != "partner-bulk-1" {
		t.Errorf("marshaled request = %v, want bulkId/partnerBulkId present", wire)
	}
	bulkObject, ok := wire["bulkObject"].([]any)
	if !ok || len(bulkObject) != 1 {
		t.Fatalf("marshaled request bulkObject = %v, want a 1-element array", wire["bulkObject"])
	}
	item, ok := bulkObject[0].(map[string]any)
	if !ok || item["customerNumber"] != "98765" || item["responseCode"] != "2004100" || item["responseMessage"] != "Success" {
		t.Errorf("marshaled request bulkObject[0] = %v, want the test's item", bulkObject[0])
	}
	if item["customerName"] != "Jane Doe" {
		t.Errorf(`marshaled request bulkObject[0]["customerName"] = %v, want "Jane Doe"`, item["customerName"])
	}
	amount, ok := item["amount"].(map[string]any)
	if !ok || amount["value"] != "100000.00" || amount["currency"] != "IDR" {
		t.Errorf(`marshaled request bulkObject[0]["amount"] = %v, want {"value":"100000.00","currency":"IDR"}`, item["amount"])
	}
	additionalInfo, ok := item["additionalInfo"].(map[string]any)
	if !ok || additionalInfo["channel"] != "mobilephone" {
		t.Errorf(`marshaled request bulkObject[0]["additionalInfo"] = %v, want {"channel":"mobilephone"}`, item["additionalInfo"])
	}

	// A zero-value marshal is the actual omitempty guard: a
	// fully-populated marshal above would still pass even if a
	// mandatory field regressed to carrying omitempty. BulkObject is a
	// nil slice without omitempty, which encoding/json marshals as the
	// JSON literal null, not an empty array — this pins that actual
	// wire behavior, mirroring
	// InterbankBulkTransferNotificationRequest.BulkObject's same edge
	// case (Phase 12).
	zb, err := json.Marshal(NotifyBulkCashInRequest{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var zeroWire map[string]any
	if err := json.Unmarshal(zb, &zeroWire); err != nil {
		t.Fatalf("decode marshaled zero-value request: %v", err)
	}
	for _, key := range []string{"bulkId", "partnerBulkId"} {
		v, ok := zeroWire[key]
		if !ok {
			t.Errorf(`zero-value marshaled request missing %q key; want it always present, even as ""`, key)
			continue
		}
		if v != "" {
			t.Errorf(`zero-value marshaled request[%q] = %v, want ""`, key, v)
		}
	}
	zeroBulkObject, ok := zeroWire["bulkObject"]
	if !ok {
		t.Fatal(`zero-value marshaled request missing "bulkObject" key; BulkObject lacks omitempty and must always be present`)
	}
	if zeroBulkObject != nil {
		t.Errorf(`zero-value marshaled request["bulkObject"] = %v, want null (nil slice without omitempty)`, zeroBulkObject)
	}
}

func TestNotifyBulkCashInResponse_RoundTrips(t *testing.T) {
	const fixture = `{
   "responseCode":"2004100",
   "responseMessage":"Request has been processed successfully",
   "bulkId":"BULK000001",
   "partnerBulkId":"partner-bulk-1"
}`
	var got NotifyBulkCashInResponse
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := NotifyBulkCashInResponse{
		ResponseCode:    "2004100",
		ResponseMessage: "Request has been processed successfully",
		BulkID:          "BULK000001",
		PartnerBulkID:   "partner-bulk-1",
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
		t.Fatalf("decode marshaled response: %v", err)
	}
	if wire["responseCode"] != "2004100" || wire["responseMessage"] != "Request has been processed successfully" ||
		wire["bulkId"] != "BULK000001" || wire["partnerBulkId"] != "partner-bulk-1" {
		t.Errorf("marshaled response = %v, want every field present", wire)
	}
}

// TestNotifyBulkCashInResponse_MarshalsBulkIDAsCamelCase pins the
// deliberate camelCase "bulkId" casing decision on this type's BulkID
// field, distinct from SubmitBulkCashInResponse's lowercase-d
// "bulkid" — see the type's doc comment for the unresolved research
// contradiction this reflects. encoding/json matches tag keys
// case-insensitively on decode, so nothing else in the package would
// catch a future edit accidentally "fixing" this tag to lowercase-d;
// only a marshal-based assertion observes the literal casing.
func TestNotifyBulkCashInResponse_MarshalsBulkIDAsCamelCase(t *testing.T) {
	b, err := json.Marshal(NotifyBulkCashInResponse{BulkID: "BULK000001"})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var got map[string]json.RawMessage
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled bytes: %v", err)
	}
	if _, ok := got["bulkId"]; !ok {
		t.Errorf(`marshaled NotifyBulkCashInResponse missing "bulkId" (camelCase) key; got keys %v`, got)
	}
	if _, ok := got["bulkid"]; ok {
		t.Error(`marshaled NotifyBulkCashInResponse has "bulkid" (lowercase d) key, want only camelCase "bulkId"`)
	}
}
