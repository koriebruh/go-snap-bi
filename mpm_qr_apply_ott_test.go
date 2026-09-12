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

func TestApplyOTT_ParsesResponse(t *testing.T) {
	const fixture = `{
   "responseCode":"2004900",
   "responseMessage":"Request has been processed successfully",
   "userResources":[
     {"resourceType":"OTT","value":"abc123token"}
   ]
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/apply-ott"
	tr := &Transport{}
	resp, err := ApplyOTT(context.Background(), tr, hb, ApplyOTTRequest{"OTT"})
	if err != nil {
		t.Fatalf("ApplyOTT() error = %v", err)
	}

	want := ApplyOTTResponse{
		ResponseCode:    "2004900",
		ResponseMessage: "Request has been processed successfully",
		UserResources: []ApplyOTTUserResource{
			{ResourceType: "OTT", Value: "abc123token"},
		},
	}
	if !reflect.DeepEqual(resp, want) {
		t.Errorf("ApplyOTT() = %+v, want %+v", resp, want)
	}
}

// TestApplyOTT_RequestBodyIsBareArray pins the phase's central,
// unresolved contradiction: the wire body is the bare JSON array
// itself, matching the source research's worked example ["OTT"], not
// an object wrapping a userResources key.
func TestApplyOTT_RequestBodyIsBareArray(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2004900","responseMessage":"ok","userResources":[]}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/apply-ott"
	tr := &Transport{}
	if _, err := ApplyOTT(context.Background(), tr, hb, ApplyOTTRequest{"OTT"}); err != nil {
		t.Fatalf("ApplyOTT() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if string(gotBody) != `["OTT"]` {
		t.Errorf(`wire body = %s, want ["OTT"] as a bare array (not {"userResources":[...]})`, gotBody)
	}
	var got []string
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("wire body = %s, want a top-level JSON array, got decode error: %v", gotBody, err)
	}
	want := []string{"OTT"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wire body = %v, want %v", got, want)
	}
}

// TestApplyOTT_NilRequestMarshalsToJSONNull pins the documented,
// unguarded behavior of a nil ApplyOTTRequest: it marshals to the JSON
// literal null, not an empty array. The package does no client-side
// validation, so this is recorded as caller responsibility, not
// prevented here.
func TestApplyOTT_NilRequestMarshalsToJSONNull(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"responseCode":"2004900","responseMessage":"ok","userResources":[]}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/apply-ott"
	tr := &Transport{}
	if _, err := ApplyOTT(context.Background(), tr, hb, nil); err != nil {
		t.Fatalf("ApplyOTT() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if string(gotBody) != "null" {
		t.Errorf(`wire body = %s, want "null" (documented nil-slice marshal behavior)`, gotBody)
	}
}

// TestApplyOTTResponse_UserResourcesHasNoOmitempty pins that
// UserResources — Mandatory per research §5.9 line 196 — has no
// omitempty tag, matching the package's mandatory-array convention.
func TestApplyOTTResponse_UserResourcesHasNoOmitempty(t *testing.T) {
	b, err := json.Marshal(ApplyOTTResponse{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value response: %v", err)
	}
	if _, ok := got["userResources"]; !ok {
		t.Fatal(`marshaled zero-value response missing "userResources" key; UserResources lacks omitempty and must always be present`)
	}
}

// TestApplyOTTUserResource_FieldsHaveNoOmitempty pins that both
// ResourceType and Value on the response item type always serialize.
func TestApplyOTTUserResource_FieldsHaveNoOmitempty(t *testing.T) {
	b, err := json.Marshal(ApplyOTTUserResource{})
	if err != nil {
		t.Fatalf("json.Marshal(zero value) error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("decode marshaled zero-value ApplyOTTUserResource: %v", err)
	}
	for _, key := range []string{"resourceType", "value"} {
		if _, ok := got[key]; !ok {
			t.Errorf(`marshaled zero-value ApplyOTTUserResource missing %q key; want it always present`, key)
		}
	}
}

func TestApplyOTT_NonTwoXXResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"responseCode":"4004900","responseMessage":"Bad Request"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/apply-ott"
	tr := &Transport{}
	_, err := ApplyOTT(context.Background(), tr, hb, ApplyOTTRequest{"OTT"})
	if err == nil {
		t.Fatal("ApplyOTT() error = nil, want non-nil for a non-2xx responseCode")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("ApplyOTT() error = %v, want errors.Is(err, ErrBadRequest)", err)
	}
}

func TestApplyOTT_NonTwoXXStatusWithTwoXXBodyIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"responseCode":"2004900","responseMessage":"ok"}`))
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/apply-ott"
	tr := &Transport{}
	resp, err := ApplyOTT(context.Background(), tr, hb, ApplyOTTRequest{"OTT"})
	if err == nil {
		t.Fatalf("ApplyOTT() error = nil, want non-nil for HTTP 500 with a 2xx-shaped body; got %+v", resp)
	}
	if !errors.Is(err, ErrInternalServerError) {
		t.Errorf("ApplyOTT() error = %v, want errors.Is(err, ErrInternalServerError)", err)
	}
}

func TestApplyOTT_TwoXXStatusWithNoResponseCodeIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"userResources":[]}`)) // valid JSON, no responseCode field
	}))
	defer server.Close()

	hb := testHeaderBuilder(server.URL)
	hb.EndpointURL = server.URL + "/v1.0/qr/apply-ott"
	tr := &Transport{}
	resp, err := ApplyOTT(context.Background(), tr, hb, ApplyOTTRequest{"OTT"})
	if err == nil {
		t.Fatalf("ApplyOTT() error = nil, want non-nil; got zero-value response = %+v", resp)
	}
}
