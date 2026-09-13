package registration

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	snap "github.com/koriebruh/go-snap-bi"
	"github.com/koriebruh/go-snap-bi/internal/snaptest"
)

func TestGetOAuthURLResponse_FieldCounts(t *testing.T) {
	if n := reflect.TypeOf(GetOAuthURLResponse{}).NumField(); n != 4 {
		t.Errorf("GetOAuthURLResponse has %d fields, want 4", n)
	}
}

func boolPtr(b bool) *bool { return &b }

func TestGetOAuthURL_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2001000",
   "responseMessage":"Request has been processed successfully",
   "authCode":"a4sd5a4fsaf5d5f4df66ad85f4",
   "state":"WodkkwijSDs"
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/get-auth-code"
	tr := &snap.Transport{}
	resp, err := GetOAuthURL(context.Background(), tr, hb, GetOAuthURLRequest{
		RedirectURL: "https://domain.com/authSuccess.htm",
		Scopes:      []string{"QUERY_BALANCE", "PUBLIC_ID"},
		State:       "WodkkwijSDs",
	})
	if err != nil {
		t.Fatalf("GetOAuthURL() error = %v", err)
	}

	want := GetOAuthURLResponse{
		ResponseCode:    "2001000",
		ResponseMessage: "Request has been processed successfully",
		AuthCode:        "a4sd5a4fsaf5d5f4df66ad85f4",
		State:           "WodkkwijSDs",
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("GetOAuthURL() = %+v, want %+v", resp, want)
	}
}

// TestGetOAuthURL_QueryStringRoundTrips pins that every field lands in
// the request's query string under the documented parameter name, and
// that Optional/Conditional fields left zero-valued are omitted
// entirely rather than sent as empty strings.
func TestGetOAuthURL_QueryStringRoundTrips(t *testing.T) {
	var gotQuery url.Values
	var gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"responseCode":"2001000","responseMessage":"ok","authCode":"a","state":"s"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/get-auth-code"
	tr := &snap.Transport{}
	req := GetOAuthURLRequest{
		RedirectURL:       "https://domain.com/authSuccess.htm",
		Scopes:            []string{"QUERY_BALANCE", "PUBLIC_ID"},
		State:             "WodkkwijSDs",
		MerchantID:        "MERCHANT001",
		SubMerchantID:     "SUBMERCHANT001",
		Lang:              "id",
		AllowRegistration: boolPtr(true),
		SeamlessData:      `{"mobileNumber":"62822999999999"}`,
		MobileNumber:      "62822999999999",
		VerifiedTime:      "2020-12-18T15:55:40+07:00",
		ExternalUID:       "ext-uid-1",
		DeviceID:          "device-1",
		SeamlessSign:      "signature-value",
	}
	if _, err := GetOAuthURL(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("GetOAuthURL() error = %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Errorf("request method = %q, want GET", gotMethod)
	}
	want := url.Values{
		"redirectUrl":       {"https://domain.com/authSuccess.htm"},
		"scopes":            {"QUERY_BALANCE,PUBLIC_ID"},
		"state":             {"WodkkwijSDs"},
		"merchantId":        {"MERCHANT001"},
		"subMerchantId":     {"SUBMERCHANT001"},
		"lang":              {"id"},
		"allowRegistration": {"true"},
		"seamlessData":      {`{"mobileNumber":"62822999999999"}`},
		"mobileNumber":      {"62822999999999"},
		"verifiedTime":      {"2020-12-18T15:55:40+07:00"},
		"externalUid":       {"ext-uid-1"},
		"deviceId":          {"device-1"},
		"seamlessSign":      {"signature-value"},
	}
	if !reflect.DeepEqual(gotQuery, want) {
		t.Errorf("query = %v, want %v", gotQuery, want)
	}
}

// TestGetOAuthURL_MinimalRequestOmitsOptionalParams pins that a request
// carrying only the three Mandatory fields sends exactly those three
// query keys — no Optional/Conditional field leaks in as an empty
// string.
func TestGetOAuthURL_MinimalRequestOmitsOptionalParams(t *testing.T) {
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"responseCode":"2001000","responseMessage":"ok","authCode":"a","state":"s"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/get-auth-code"
	tr := &snap.Transport{}
	req := GetOAuthURLRequest{
		RedirectURL: "https://domain.com/authSuccess.htm",
		Scopes:      []string{"QUERY_BALANCE"},
		State:       "state-1",
	}
	if _, err := GetOAuthURL(context.Background(), tr, hb, req); err != nil {
		t.Fatalf("GetOAuthURL() error = %v", err)
	}

	want := url.Values{
		"redirectUrl": {"https://domain.com/authSuccess.htm"},
		"scopes":      {"QUERY_BALANCE"},
		"state":       {"state-1"},
	}
	if !reflect.DeepEqual(gotQuery, want) {
		t.Errorf("query = %v, want %v", gotQuery, want)
	}
}

func TestGetOAuthURL_RejectsEndpointURLWithExistingQuery(t *testing.T) {
	hb := snaptest.TestHeaderBuilder("http://example.com")
	hb.EndpointURL = "http://example.com/v1.0/get-auth-code?already=here"
	tr := &snap.Transport{}
	_, err := GetOAuthURL(context.Background(), tr, hb, GetOAuthURLRequest{
		RedirectURL: "https://domain.com/authSuccess.htm",
		Scopes:      []string{"QUERY_BALANCE"},
		State:       "state-1",
	})
	if !errors.Is(err, errGetOAuthURLInvalidEndpointURL) {
		t.Errorf("GetOAuthURL() error = %v, want errors.Is(err, errGetOAuthURLInvalidEndpointURL)", err)
	}
}

func TestGetOAuthURL_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4001000","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/get-auth-code"
	tr := &snap.Transport{}
	_, err := GetOAuthURL(context.Background(), tr, hb, GetOAuthURLRequest{
		RedirectURL: "https://domain.com/authSuccess.htm",
		Scopes:      []string{"QUERY_BALANCE"},
		State:       "state-1",
	})
	if err == nil {
		t.Fatal("GetOAuthURL() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, snap.ErrBadRequest) {
		t.Errorf("GetOAuthURL() error = %v, want errors.Is(err, snap.ErrBadRequest)", err)
	}
}

func TestGetOAuthURL_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2001000","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/get-auth-code"
	tr := &snap.Transport{}
	resp, err := GetOAuthURL(context.Background(), tr, hb, GetOAuthURLRequest{
		RedirectURL: "https://domain.com/authSuccess.htm",
		Scopes:      []string{"QUERY_BALANCE"},
		State:       "state-1",
	})
	if err == nil {
		t.Fatalf("GetOAuthURL() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, snap.ErrInternalServerError) {
		t.Errorf("GetOAuthURL() error = %v, want errors.Is(err, snap.ErrInternalServerError)", err)
	}
}

func TestGetOAuthURL_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"authCode":"a"}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := snaptest.TestHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/get-auth-code"
	tr := &snap.Transport{}
	resp, err := GetOAuthURL(context.Background(), tr, hb, GetOAuthURLRequest{
		RedirectURL: "https://domain.com/authSuccess.htm",
		Scopes:      []string{"QUERY_BALANCE"},
		State:       "state-1",
	})
	if err == nil {
		t.Fatalf("GetOAuthURL() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
