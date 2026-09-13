package transferkredit

import (
	"reflect"
	"testing"
)

// TestUpdateVAData_UpdateStatusVAData_InquiryVAData_IdenticalShape pins
// that UpdateVAData, UpdateStatusVAData, and InquiryVAData — three
// distinct Go types the design doc declares field-identical (research
// doc: "Response is the full VA data object... same shape as Update VA
// response") — actually stay identical in field name, order, Go type,
// and JSON tag. These types have no compile-time link to each other, so
// nothing else in the package would catch one of them drifting from the
// other two.
func TestUpdateVAData_UpdateStatusVAData_InquiryVAData_IdenticalShape(t *testing.T) {
	shape := func(v any) []string {
		typ := reflect.TypeOf(v)
		fields := make([]string, typ.NumField())
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			fields[i] = f.Name + " " + f.Type.String() + " `" + string(f.Tag) + "`"
		}
		return fields
	}

	want := shape(UpdateVAData{})
	for name, got := range map[string][]string{
		"UpdateStatusVAData": shape(UpdateStatusVAData{}),
		"InquiryVAData":      shape(InquiryVAData{}),
	} {
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s field shape = %v, want (matching UpdateVAData) %v", name, got, want)
		}
	}
}

// TestVAInquiryStatusData_MirrorsVAPaymentDataPlusTransactionDate pins
// that VAInquiryStatusData (Phase 14) carries every field of
// VAPaymentData (same Go type and JSON tag), per research §5.3's "VA
// Inquiry Status ... mirrors VA Payment's response shape plus
// transactionDate." Unlike UpdateVAData/UpdateStatusVAData/InquiryVAData
// (byte-identical), VAInquiryStatusData has one extra field, so this
// checks a subset relationship instead of full equality — these two
// types have no compile-time link, so nothing else would catch a
// hand-edit to one field/tag not mirrored on the other.
func TestVAInquiryStatusData_MirrorsVAPaymentDataPlusTransactionDate(t *testing.T) {
	fieldSet := func(v any) map[string]struct {
		typ string
		tag string
	} {
		typ := reflect.TypeOf(v)
		fields := make(map[string]struct {
			typ string
			tag string
		}, typ.NumField())
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			fields[f.Name] = struct {
				typ string
				tag string
			}{typ: f.Type.String(), tag: string(f.Tag)}
		}
		return fields
	}

	paymentFields := fieldSet(VAPaymentData{})
	statusFields := fieldSet(VAInquiryStatusData{})
	for name, want := range paymentFields {
		got, ok := statusFields[name]
		if !ok {
			t.Errorf("VAInquiryStatusData missing field %q present on VAPaymentData", name)
			continue
		}
		if got != want {
			t.Errorf("VAInquiryStatusData.%s = %+v, want (matching VAPaymentData) %+v", name, got, want)
		}
	}

	td, ok := statusFields["TransactionDate"]
	if !ok {
		t.Fatal("VAInquiryStatusData missing TransactionDate field")
	}
	wantTD := struct {
		typ string
		tag string
	}{typ: "string", tag: `json:"transactionDate,omitempty"`}
	if td != wantTD {
		t.Errorf("VAInquiryStatusData.TransactionDate = %+v, want %+v", td, wantTD)
	}

	if want, got := len(paymentFields)+1, len(statusFields); got != want {
		t.Errorf("VAInquiryStatusData has %d fields, want %d (VAPaymentData's %d + TransactionDate) — an unrelated field added to VAInquiryStatusData would otherwise pass this test silently", got, want, len(paymentFields))
	}
}

// TestVAIntrabankReportResponses_UseLowercaseDVirtualAccountdataTag
// pins the single highest-risk fact from the Phase 15 design doc:
// research §5.3 line 132 confirms, "not a table typo," that Service
// Codes 32-35 use the response envelope key "virtualAccountdata"
// (lowercase d), unlike every VA type from Phases 13-14, which use
// capital-D "virtualAccountData". A copy-paste of an existing
// *Response struct tag would silently produce a field that never
// populates — encoding/json ignores unknown keys with no error — and
// each endpoint's own ParsesResponse test is hand-built to mirror the
// struct, so it cannot catch this by itself. This test asserts the
// literal struct tag string on the VirtualAccountData field of all
// four new *Response types.
func TestVAIntrabankReportResponses_UseLowercaseDVirtualAccountdataTag(t *testing.T) {
	const wantSingle = `json:"virtualAccountdata,omitempty"`
	singleTagged := map[string]any{
		"VAInquiryPaymentIntrabankResponse": VAInquiryPaymentIntrabankResponse{},
		"VAPaymentIntrabankResponse":        VAPaymentIntrabankResponse{},
		"VANotifyPaymentIntrabankResponse":  VANotifyPaymentIntrabankResponse{},
	}
	for name, v := range singleTagged {
		f, ok := reflect.TypeOf(v).FieldByName("VirtualAccountData")
		if !ok {
			t.Errorf("%s has no VirtualAccountData field", name)
			continue
		}
		if got := string(f.Tag); got != wantSingle {
			t.Errorf("%s.VirtualAccountData tag = %s, want %s", name, got, wantSingle)
		}
	}

	f, ok := reflect.TypeOf(GetReportResponse{}).FieldByName("VirtualAccountData")
	if !ok {
		t.Fatal("GetReportResponse has no VirtualAccountData field")
	}
	if got := string(f.Tag); got != wantSingle {
		t.Errorf("GetReportResponse.VirtualAccountData tag = %s, want %s", got, wantSingle)
	}
	if f.Type.Kind() != reflect.Slice {
		t.Errorf("GetReportResponse.VirtualAccountData type = %s, want a slice (the only array-typed VA response in the package)", f.Type)
	}
}

// TestGetReportData_IdenticalToVAInquiryStatusData pins that
// GetReportData (Phase 15) is field-identical to VAInquiryStatusData
// (Phase 14), per research §5.3 line 155's "each item shaped like the
// Payment/Inquiry-Status response object." These two types have no
// compile-time link to each other, so nothing else in the package
// would catch one of them drifting from the other.
func TestGetReportData_IdenticalToVAInquiryStatusData(t *testing.T) {
	shape := func(v any) []string {
		typ := reflect.TypeOf(v)
		fields := make([]string, typ.NumField())
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			fields[i] = f.Name + " " + f.Type.String() + " `" + string(f.Tag) + "`"
		}
		return fields
	}

	want := shape(VAInquiryStatusData{})
	got := shape(GetReportData{})
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetReportData field shape = %v, want (matching VAInquiryStatusData) %v", got, want)
	}
}
