package snap

// TransferOriginatorInfo is the shared "originatorInfos[]" entry used by
// the Trigger Transfer sub-group. All three fields are String per the
// Guides tab.
type TransferOriginatorInfo struct {
	OriginatorCustomerNo   string `json:"originatorCustomerNo"`
	OriginatorCustomerName string `json:"originatorCustomerName"`
	OriginatorBankCode     string `json:"originatorBankCode"`
}
