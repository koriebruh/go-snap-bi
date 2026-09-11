package snap

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestInterbankBulkTransferNotificationRequest_RoundTrips(t *testing.T) {
	const fixture = `{
   "bulkId":"BULK123456",
   "partnerBulkId":"partner-bulk-1",
   "bulkObject":[{"partnerReferenceNo":"pr-1","responseCode":"2002100","responseMessage":"Success"}]
}`
	var got InterbankBulkTransferNotificationRequest
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := InterbankBulkTransferNotificationRequest{
		BulkID:        "BULK123456",
		PartnerBulkID: "partner-bulk-1",
		BulkObject: []InterbankBulkTransferNotificationItem{
			{PartnerReferenceNo: "pr-1", ResponseCode: "2002100", ResponseMessage: "Success"},
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
	if wire["bulkId"] != "BULK123456" || wire["partnerBulkId"] != "partner-bulk-1" {
		t.Errorf("marshaled request = %v, want bulkId/partnerBulkId present", wire)
	}
	bulkObject, ok := wire["bulkObject"].([]any)
	if !ok || len(bulkObject) != 1 {
		t.Fatalf("marshaled request bulkObject = %v, want a 1-element array", wire["bulkObject"])
	}
	item, ok := bulkObject[0].(map[string]any)
	if !ok || item["partnerReferenceNo"] != "pr-1" || item["responseCode"] != "2002100" || item["responseMessage"] != "Success" {
		t.Errorf("marshaled request bulkObject[0] = %v, want the test's item", bulkObject[0])
	}

	// A zero-value marshal is the actual omitempty guard: a fully-populated
	// marshal above would still pass even if a mandatory field regressed to
	// carrying omitempty. BulkObject is a nil slice without omitempty, which
	// encoding/json marshals as the JSON literal null, not an empty array —
	// this pins that actual wire behavior, mirroring
	// InterbankBulkTransferRequest.BulkObject's same edge case.
	zb, err := json.Marshal(InterbankBulkTransferNotificationRequest{})
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

func TestInterbankBulkTransferNotificationResponse_RoundTrips(t *testing.T) {
	const fixture = `{
   "responseCode":"2002100",
   "responseMessage":"Request has been processed successfully",
   "bulkId":"BULK123456",
   "partnerBulkId":"partner-bulk-1",
   "additionalInfo":{"channel":"mobilephone"}
}`
	var got InterbankBulkTransferNotificationResponse
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := InterbankBulkTransferNotificationResponse{
		ResponseCode:    "2002100",
		ResponseMessage: "Request has been processed successfully",
		BulkID:          "BULK123456",
		PartnerBulkID:   "partner-bulk-1",
		AdditionalInfo:  json.RawMessage(`{"channel":"mobilephone"}`),
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
	if wire["responseCode"] != "2002100" || wire["responseMessage"] != "Request has been processed successfully" ||
		wire["bulkId"] != "BULK123456" || wire["partnerBulkId"] != "partner-bulk-1" {
		t.Errorf("marshaled response = %v, want every field present", wire)
	}
}
