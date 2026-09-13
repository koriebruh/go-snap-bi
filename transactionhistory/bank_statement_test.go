package transactionhistory

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

func TestBankStatementTypes_FieldCounts(t *testing.T) {
	cases := []struct {
		name string
		typ  reflect.Type
		want int
	}{
		{"BankStatementRequest", reflect.TypeOf(BankStatementRequest{}), 6},
		{"BankStatementResponse", reflect.TypeOf(BankStatementResponse{}), 11},
		{"BankStatementBalance", reflect.TypeOf(BankStatementBalance{}), 3},
		{"BankStatementBalanceAmount", reflect.TypeOf(BankStatementBalanceAmount{}), 3},
		{"BankStatementEntryTotal", reflect.TypeOf(BankStatementEntryTotal{}), 2},
		{"BankStatementDetail", reflect.TypeOf(BankStatementDetail{}), 9},
		{"BankStatementDetailBalance", reflect.TypeOf(BankStatementDetailBalance{}), 2},
		{"BankStatementDetailBalanceEntry", reflect.TypeOf(BankStatementDetailBalanceEntry{}), 1},
	}
	for _, c := range cases {
		if n := c.typ.NumField(); n != c.want {
			t.Errorf("%s has %d fields, want %d", c.name, n, c.want)
		}
	}
}

func TestBankStatementRequest_RoundTrips(t *testing.T) {
	const fixture = `{
   "partnerReferenceNo":"2020102900000000000001",
   "bankCardToken":"6d7963617264746f6b656e",
   "accountNo":"7382382957893840",
   "fromDateTime":"2019-07-03T12:08:56+07:00",
   "toDateTime":"2019-07-03T12:08:56+07:00",
   "additionalInfo":{"deviceId":"12345679237","channel":"mobilephone"}
}`
	var got BankStatementRequest
	if err := json.Unmarshal([]byte(fixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := BankStatementRequest{
		PartnerReferenceNo: "2020102900000000000001",
		BankCardToken:      "6d7963617264746f6b656e",
		AccountNo:          "7382382957893840",
		FromDateTime:       "2019-07-03T12:08:56+07:00",
		ToDateTime:         "2019-07-03T12:08:56+07:00",
		AdditionalInfo:     json.RawMessage(`{"deviceId":"12345679237","channel":"mobilephone"}`),
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
		"partnerReferenceNo": "2020102900000000000001",
		"bankCardToken":      "6d7963617264746f6b656e",
		"accountNo":          "7382382957893840",
		"fromDateTime":       "2019-07-03T12:08:56+07:00",
		"toDateTime":         "2019-07-03T12:08:56+07:00",
		"additionalInfo":     map[string]any{"deviceId": "12345679237", "channel": "mobilephone"},
	}
	if !reflect.DeepEqual(wire, wantWire) {
		t.Errorf("marshaled request = %v, want %v", wire, wantWire)
	}
}

// TestBankStatementRequest_MandatoryFieldsHaveNoOmitempty pins that every
// BankStatementRequest field is Optional/Conditional and so a zero-value
// request marshals to an empty object.
func TestBankStatementRequest_MandatoryFieldsHaveNoOmitempty(t *testing.T) {
	b, err := json.Marshal(BankStatementRequest{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value request: %v", err)
	}
	want := map[string]any{}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value request = %v, want %v (every field Optional/Conditional)", got, want)
	}
}

func TestBankStatementRequest_MalformedAdditionalInfoIsMarshalError(t *testing.T) {
	req := BankStatementRequest{
		PartnerReferenceNo: "2020102900000000000001",
		AdditionalInfo:     json.RawMessage(`{not-valid-json`),
	}
	if _, err := json.Marshal(req); err == nil {
		t.Error("json.Marshal() error = nil, want an error for malformed AdditionalInfo")
	}
}

func fullBankStatementResponse() BankStatementResponse {
	return BankStatementResponse{
		ResponseCode:       "2001400",
		ResponseMessage:    "Request has been processed successfully",
		ReferenceNo:        "2020102977770000000009",
		PartnerReferenceNo: "2020102900000000000001",
		Balance: []BankStatementBalance{
			{
				Amount:          BankStatementBalanceAmount{Value: "10000.00", Currency: "IDR", DateTime: "2020-12-18T16:03:45+07:00"},
				StartingBalance: BankStatementBalanceAmount{Value: "12345678.00", Currency: "IDR", DateTime: "2020-12-18T16:03:45+07:00"},
				EndingBalance:   BankStatementBalanceAmount{Value: "12345678.00", Currency: "IDR", DateTime: "2020-12-19T16:03:45+07:00"},
			},
		},
		TotalCreditEntries: &BankStatementEntryTotal{
			NumberOfEntries: "10",
			Amount:          snap.Money{Value: "10000.00", Currency: "IDR"},
		},
		TotalDebitEntries: &BankStatementEntryTotal{
			NumberOfEntries: "10",
			Amount:          snap.Money{Value: "10000.00", Currency: "IDR"},
		},
		HasMore:            "Y",
		LastRecordDateTime: "2021-11-24T11:54:56+07:00",
		DetailData: []BankStatementDetail{
			{
				DetailBalance: &BankStatementDetailBalance{
					StartAmount: []BankStatementDetailBalanceEntry{{Amount: &snap.Money{Value: "10000.00", Currency: "IDR"}}},
					EndAmount:   []BankStatementDetailBalanceEntry{{Amount: &snap.Money{Value: "10000.00", Currency: "IDR"}}},
				},
				Amount:                  &snap.Money{Value: "12345678.00", Currency: "IDR"},
				OriginAmount:            &snap.Money{Value: "12345678.00", Currency: "IDR"},
				TransactionDate:         "2009-07-03T12:08:56+07:00",
				Remark:                  "Payment to Warung Ikan Bakar",
				TransactionID:           "20200801198230912830091123",
				Type:                    "CREDIT",
				TransactionDetailStatus: "SUCCESS",
				DetailInfo:              json.RawMessage(`{"page":"12"}`),
			},
		},
		AdditionalInfo: json.RawMessage(`{"deviceId":"12345679237","channel":"mobilephone"}`),
	}
}

const fullBankStatementResponseFixture = `{
   "responseCode":"2001400",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"2020102977770000000009",
   "partnerReferenceNo":"2020102900000000000001",
   "balance":[{
      "amount":{"value":"10000.00","currency":"IDR","dateTime":"2020-12-18T16:03:45+07:00"},
      "startingBalance":{"value":"12345678.00","currency":"IDR","dateTime":"2020-12-18T16:03:45+07:00"},
      "endingBalance":{"value":"12345678.00","currency":"IDR","dateTime":"2020-12-19T16:03:45+07:00"}
   }],
   "totalCreditEntries":{"numberOfEntries":"10","amount":{"value":"10000.00","currency":"IDR"}},
   "totalDebitEntries":{"numberOfEntries":"10","amount":{"value":"10000.00","currency":"IDR"}},
   "hasMore":"Y",
   "lastRecordDateTime":"2021-11-24T11:54:56+07:00",
   "detailData":[{
      "detailBalance":{
         "startAmount":[{"amount":{"value":"10000.00","currency":"IDR"}}],
         "endAmount":[{"amount":{"value":"10000.00","currency":"IDR"}}]
      },
      "amount":{"value":"12345678.00","currency":"IDR"},
      "originAmount":{"value":"12345678.00","currency":"IDR"},
      "transactionDate":"2009-07-03T12:08:56+07:00",
      "remark":"Payment to Warung Ikan Bakar",
      "transactionId":"20200801198230912830091123",
      "type":"CREDIT",
      "transactionDetailStatus":"SUCCESS",
      "detailInfo":{"page":"12"}
   }],
   "additionalInfo":{"deviceId":"12345679237","channel":"mobilephone"}
}`

func TestBankStatementResponse_RoundTrips(t *testing.T) {
	var got BankStatementResponse
	if err := json.Unmarshal([]byte(fullBankStatementResponseFixture), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := fullBankStatementResponse()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("json.Unmarshal() = %+v, want %+v", got, want)
	}
}

// TestBankStatementResponse_MandatoryFieldsHaveNoOmitempty pins
// ResponseCode and ResponseMessage as the only always-serializing fields
// from a zero-value response.
func TestBankStatementResponse_MandatoryFieldsHaveNoOmitempty(t *testing.T) {
	b, err := json.Marshal(BankStatementResponse{})
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
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marshaled zero-value response = %v, want %v", got, want)
	}
}

func TestBankStatement_ParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fullBankStatementResponseFixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/bank-statement"
	tr := &snap.Transport{}
	resp, err := BankStatement(context.Background(), tr, hb, BankStatementRequest{
		PartnerReferenceNo: "2020102900000000000001",
		AccountNo:          "7382382957893840",
	})
	if err != nil {
		t.Fatalf("BankStatement() error = %v", err)
	}

	want := fullBankStatementResponse()
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("BankStatement() = %+v, want %+v", resp, want)
	}
}

func TestBankStatement_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2001400","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/bank-statement"
	tr := &snap.Transport{}
	req := BankStatementRequest{
		PartnerReferenceNo: "2020102900000000000001",
		AccountNo:          "7382382957893840",
		FromDateTime:       "2019-07-03T12:08:56+07:00",
		ToDateTime:         "2019-07-03T12:08:56+07:00",
	}
	if _, err := BankStatement(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("BankStatement() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["accountNo"] != "7382382957893840" {
		t.Errorf(`wire body["accountNo"] = %v, want "7382382957893840"`, got["accountNo"])
	}
	if _, ok := got["bankCardToken"]; ok {
		t.Errorf(`wire body["bankCardToken"] = %v, want absent (omitempty, unset)`, got["bankCardToken"])
	}
}

func TestBankStatement_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4001400","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/bank-statement"
	tr := &snap.Transport{}
	_, err := BankStatement(context.Background(), tr, hb, BankStatementRequest{
		PartnerReferenceNo: "2020102900000000000001",
		AccountNo:          "7382382957893840",
	})
	if err == nil {
		t.Fatal("BankStatement() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("BankStatement() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestBankStatement_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2001400","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/bank-statement"
	tr := &snap.Transport{}
	resp, err := BankStatement(context.Background(), tr, hb, BankStatementRequest{
		PartnerReferenceNo: "2020102900000000000001",
		AccountNo:          "7382382957893840",
	})
	if err == nil {
		t.Fatalf("BankStatement() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("BankStatement() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestBankStatement_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"referenceNo":"2020102977770000000009"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/bank-statement"
	tr := &snap.Transport{}
	resp, err := BankStatement(context.Background(), tr, hb, BankStatementRequest{
		PartnerReferenceNo: "2020102900000000000001",
		AccountNo:          "7382382957893840",
	})
	if err == nil {
		t.Fatalf("BankStatement() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
