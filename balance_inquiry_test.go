package snap

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// standardWorkedExampleBalanceInquiryResponse is the standard's own worked
// example response body (Code Snippets tab, ASPI SNAP Developer Site),
// hardcoded independently of this package's own types so a broken
// implementation can't tautologically agree with itself.
const standardWorkedExampleBalanceInquiryResponse = `{
   "responseCode":"2001100",
   "responseMessage":"Request has been processed successfully",
   "referenceNo":"2020102977770000000009",
   "partnerReferenceNo":"2020102900000000000001",
   "accountNo":"115471119",
   "name":"JONOMADE",
   "accountInfos":[
      {
         "balanceType":"Cash",
         "amount":{"value":"200000.00","currency":"IDR"},
         "floatAmount":{"value":"50000.00","currency":"IDR"},
         "holdAmount":{"value":"20000.00","currency":"IDR"},
         "availableBalance":{"value":"130000.00","currency":"IDR"},
         "ledgerBalance":{"value":"30000.00","currency":"IDR"},
         "currentMultilateralLimit":{"value":"10000.00","currency":"IDR"},
         "registrationStatusCode":"0001",
         "status":"0001"
      }
   ]
}`

func testHeaderBuilder(serverURL string) HeaderBuilder {
	return HeaderBuilder{
		Method:       http.MethodPost,
		EndpointURL:  serverURL + "/v1.0/balance-inquiry",
		AccessToken:  "access-token-123",
		ClientKey:    "client-key",
		PartnerID:    "partner-id",
		ExternalID:   "external-id",
		ChannelID:    "channel-id",
		Symmetric:    true,
		ClientSecret: "shh-secret",
	}
}

func TestBalanceInquiry_ParsesWorkedExampleResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(standardWorkedExampleBalanceInquiryResponse))
	}))
	defer server.Close()

	tr := &Transport{}
	resp, err := BalanceInquiry(context.Background(), tr, testHeaderBuilder(server.URL), BalanceInquiryRequest{
		PartnerReferenceNo: "2020102900000000000001",
		AccountNo:          "7382382957893840",
	})
	if err != nil {
		t.Fatalf("BalanceInquiry() error = %v", err)
	}

	if resp.ResponseCode != "2001100" {
		t.Errorf("ResponseCode = %q, want %q", resp.ResponseCode, "2001100")
	}
	if resp.AccountNo != "115471119" {
		t.Errorf("AccountNo = %q, want %q", resp.AccountNo, "115471119")
	}
	if resp.Name != "JONOMADE" {
		t.Errorf("Name = %q, want %q", resp.Name, "JONOMADE")
	}
	if len(resp.AccountInfos) != 1 {
		t.Fatalf("len(AccountInfos) = %d, want 1", len(resp.AccountInfos))
	}
	info := resp.AccountInfos[0]
	if info.BalanceType != "Cash" {
		t.Errorf("AccountInfos[0].BalanceType = %q, want %q", info.BalanceType, "Cash")
	}
	if info.Amount != (Money{Value: "200000.00", Currency: "IDR"}) {
		t.Errorf("AccountInfos[0].Amount = %+v, want {200000.00 IDR}", info.Amount)
	}
	if info.AvailableBalance != (Money{Value: "130000.00", Currency: "IDR"}) {
		t.Errorf("AccountInfos[0].AvailableBalance = %+v, want {130000.00 IDR}", info.AvailableBalance)
	}
	if info.Status != "0001" {
		t.Errorf("AccountInfos[0].Status = %q, want %q", info.Status, "0001")
	}
}

func TestBalanceInquiry_RequestBodyRoundTrips(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2001100","responseMessage":"ok"}`))
	}))
	defer server.Close()

	tr := &Transport{}
	req := BalanceInquiryRequest{
		PartnerReferenceNo: "ref-1",
		AccountNo:          "123456",
		BalanceTypes:       []string{"Cash", "Coins"},
	}
	if _, err := BalanceInquiry(context.Background(), tr, testHeaderBuilder(server.URL), req); err != nil {
		t.Fatalf("BalanceInquiry() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var got BalanceInquiryRequest
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got.PartnerReferenceNo != req.PartnerReferenceNo || got.AccountNo != req.AccountNo {
		t.Errorf("server received %+v, want partnerReferenceNo/accountNo matching %+v", got, req)
	}
	if len(got.BalanceTypes) != 2 || got.BalanceTypes[0] != "Cash" || got.BalanceTypes[1] != "Coins" {
		t.Errorf("server received BalanceTypes = %v, want [Cash Coins]", got.BalanceTypes)
	}
}

func TestBalanceInquiry_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4001100","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	tr := &Transport{}
	_, err := BalanceInquiry(context.Background(), tr, testHeaderBuilder(server.URL), BalanceInquiryRequest{AccountNo: "123"})
	if err == nil {
		t.Fatal("BalanceInquiry() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("BalanceInquiry() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}
