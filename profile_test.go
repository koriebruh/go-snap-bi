package snap

import "testing"

func TestDefaultProfile_BuildPath(t *testing.T) {
	tests := []struct {
		name         string
		profile      DefaultProfile
		serviceGroup string
		productType  string
		want         string
	}{
		{
			name:         "explicit version",
			profile:      DefaultProfile{Domain: "openapi.example.com", Version: "v1.0"},
			serviceGroup: "payment-transfer",
			productType:  "transfer-va",
			want:         "/openapi.example.com/v1.0/payment-transfer/transfer-va",
		},
		{
			name:         "different service group and product type",
			profile:      DefaultProfile{Domain: "snap.bank.co.id", Version: "v1.0"},
			serviceGroup: "balance-inquiry",
			productType:  "account-balance",
			want:         "/snap.bank.co.id/v1.0/balance-inquiry/account-balance",
		},
		{
			name:         "empty version defaults to v1.0",
			profile:      DefaultProfile{Domain: "openapi.example.com"},
			serviceGroup: "qris",
			productType:  "qr-mpm-generate",
			want:         "/openapi.example.com/v1.0/qris/qr-mpm-generate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.profile.BuildPath(tt.serviceGroup, tt.productType)
			if got != tt.want {
				t.Errorf("BuildPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDefaultProfile_TimestampLayout(t *testing.T) {
	p := DefaultProfile{}
	if got := p.TimestampLayout(); got != DefaultTimestampLayout {
		t.Errorf("TimestampLayout() = %q, want %q", got, DefaultTimestampLayout)
	}
}
