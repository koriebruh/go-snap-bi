package snap

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
