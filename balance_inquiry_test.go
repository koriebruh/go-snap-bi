package snap

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
         "status":"0001",
         "additionalInfo":{"note":"per-account"}
      }
   ],
   "additionalInfo":{"deviceId":"12345679237","channel":"mobilephone"}
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

	// Compare the entire decoded response against a literal expected value —
	// not a hand-picked subset — so a typo'd json tag on any field,
	// including ResponseMessage and AdditionalInfo, can't pass silently.
	want := BalanceInquiryResponse{
		ResponseCode:       "2001100",
		ResponseMessage:    "Request has been processed successfully",
		ReferenceNo:        "2020102977770000000009",
		PartnerReferenceNo: "2020102900000000000001",
		AccountNo:          "115471119",
		Name:               "JONOMADE",
		AccountInfos: []AccountInfo{
			{
				BalanceType:              "Cash",
				Amount:                   Money{Value: "200000.00", Currency: "IDR"},
				FloatAmount:              Money{Value: "50000.00", Currency: "IDR"},
				HoldAmount:               Money{Value: "20000.00", Currency: "IDR"},
				AvailableBalance:         Money{Value: "130000.00", Currency: "IDR"},
				LedgerBalance:            Money{Value: "30000.00", Currency: "IDR"},
				CurrentMultilateralLimit: Money{Value: "10000.00", Currency: "IDR"},
				RegistrationStatusCode:   "0001",
				Status:                   "0001",
				AdditionalInfo:           json.RawMessage(`{"note":"per-account"}`),
			},
		},
		AdditionalInfo: json.RawMessage(`{"deviceId":"12345679237","channel":"mobilephone"}`),
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("BalanceInquiry() = %+v, want %+v", resp, want)
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
		AdditionalInfo:     json.RawMessage(`{"channel":"mobilephone"}`),
	}
	if _, err := BalanceInquiry(context.Background(), tr, testHeaderBuilder(server.URL), req); err != nil {
		t.Fatalf("BalanceInquiry() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	// Decode into map[string]any and check the actual wire key names,
	// rather than unmarshaling into BalanceInquiryRequest again — that
	// would make a wrong json tag on the request side (e.g. accountNo ->
	// account_no) marshal and unmarshal consistently with itself and still
	// pass.
	var got map[string]any
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("decode request body the server received: %v", err)
	}
	if got["partnerReferenceNo"] != "ref-1" {
		t.Errorf(`wire body["partnerReferenceNo"] = %v, want "ref-1"`, got["partnerReferenceNo"])
	}
	if got["accountNo"] != "123456" {
		t.Errorf(`wire body["accountNo"] = %v, want "123456"`, got["accountNo"])
	}
	balanceTypes, ok := got["balanceTypes"].([]any)
	if !ok || len(balanceTypes) != 2 || balanceTypes[0] != "Cash" || balanceTypes[1] != "Coins" {
		t.Errorf(`wire body["balanceTypes"] = %v, want ["Cash" "Coins"]`, got["balanceTypes"])
	}
	additionalInfo, ok := got["additionalInfo"].(map[string]any)
	if !ok || additionalInfo["channel"] != "mobilephone" {
		t.Errorf(`wire body["additionalInfo"] = %v, want {"channel":"mobilephone"}`, got["additionalInfo"])
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

// TestBalanceInquiry_NonTwoXXWithNoResponseCodeIsError is the regression
// test for a go-review finding (and a santa-loop finding that the fix
// should be errors.Is-matchable via env.StatusCode, not just non-nil): a
// non-2xx HTTP response whose body doesn't carry a responseCode field at
// all (e.g. a proxy/WAF error page) was previously returned as a
// "successful" zero-value BalanceInquiryResponse with a nil error.
func TestBalanceInquiry_NonTwoXXWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"boom"}`)) // no responseCode field at all
	}))
	defer server.Close()

	tr := &Transport{}
	resp, err := BalanceInquiry(context.Background(), tr, testHeaderBuilder(server.URL), BalanceInquiryRequest{AccountNo: "123"})
	if err == nil {
		t.Fatalf("BalanceInquiry() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("BalanceInquiry() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

// TestBalanceInquiry_TwoXXStatusWithNoResponseCodeIsError is the binding-
// level regression test for a santa-loop finding: the "HTTP 200, but the
// JSON body itself carries no responseCode" path was previously reachable
// only in theory — every other no-responseCode test in the package pairs
// it with a non-2xx HTTP status, so this exact branch (the final guard in
// BalanceInquiry, checkResponseStatus's 2xx passthrough) had never once
// been exercised at HTTP 200.
func TestBalanceInquiry_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// No WriteHeader call: httptest defaults to 200.
		_, _ = w.Write([]byte(`{"accountNo":"123"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	tr := &Transport{}
	resp, err := BalanceInquiry(context.Background(), tr, testHeaderBuilder(server.URL), BalanceInquiryRequest{AccountNo: "123"})
	if err == nil {
		t.Fatalf("BalanceInquiry() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}

// TestBalanceInquiry_NonTwoXXStatusWithTwoXXBodyIsError is the regression
// test for a santa-loop round-3 HIGH finding: an HTTP 500 (or any non-2xx
// status) whose body claims a 2xx-class responseCode — e.g. a stale cached
// body from a misbehaving intermediary — used to be returned as a
// successful, fully-populated response, since envelopeError only inspects
// the code's own embedded status, never the actual transport status.
func TestBalanceInquiry_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(standardWorkedExampleBalanceInquiryResponse)) // responseCode "2001100"
	}))
	defer server.Close()

	tr := &Transport{}
	resp, err := BalanceInquiry(context.Background(), tr, testHeaderBuilder(server.URL), BalanceInquiryRequest{AccountNo: "123"})
	if err == nil {
		t.Fatalf("BalanceInquiry() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("BalanceInquiry() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

// TestBalanceInquiry_UnmappedStatusIsErrorsIsMatchable is the regression
// test for the exported ErrUnmappedResponseCode: a gateway status SNAP's
// own table doesn't document (502, the canonical proxy/gateway error) must
// still be errors.Is-matchable, not just a non-nil opaque error.
func TestBalanceInquiry_UnmappedStatusIsErrorsIsMatchable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>502 Bad Gateway</html>"))
	}))
	defer server.Close()

	tr := &Transport{}
	_, err := BalanceInquiry(context.Background(), tr, testHeaderBuilder(server.URL), BalanceInquiryRequest{AccountNo: "123"})
	if err == nil {
		t.Fatal("BalanceInquiry() error = nil, want non-nil for a 502 response")
	}
	if !errors.Is(err, ErrUnmappedResponseCode) {
		t.Errorf("BalanceInquiry() error = %v, want errors.Is(err, ErrUnmappedResponseCode)", err)
	}
}

// TestBalanceInquiry_NonTwoXXWithNonJSONBodyIsError is the regression test
// for a santa-loop round-2 finding: a non-2xx response whose body isn't
// even JSON (an HTML WAF page, or an empty body from a load balancer) used
// to fail with an opaque decode error, not an errors.Is-matchable sentinel
// — Transport.Do returned before Envelope (and its StatusCode) was ever
// built, so the round-1 fix couldn't reach this case.
func TestBalanceInquiry_NonTwoXXWithNonJSONBodyIsError(t *testing.T) {
	tests := []struct {
		name       string
		httpStatus int
		body       string
		wantErr    error
	}{
		{"HTML WAF page", http.StatusUnauthorized, "<html><body>401 Unauthorized</body></html>", ErrUnauthorized},
		{"empty body", http.StatusServiceUnavailable, "", ErrServiceUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.httpStatus)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			tr := &Transport{}
			_, err := BalanceInquiry(context.Background(), tr, testHeaderBuilder(server.URL), BalanceInquiryRequest{AccountNo: "123"})
			if err == nil {
				t.Fatal("BalanceInquiry() error = nil, want non-nil")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("BalanceInquiry() error = %v, want errors.Is(err, %v)", err, tt.wantErr)
			}
		})
	}
}
