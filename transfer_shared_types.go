package snap

// TransferAmount is the shared "amount"-shaped object used across the
// Transfer Kredit group (e.g. amount, feeAmount, totalAmount). Both
// fields are String per the Guides tab.
type TransferAmount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

// TransferOriginatorInfo is the shared "originatorInfos[]" entry used by
// the Trigger Transfer sub-group. All three fields are String per the
// Guides tab.
type TransferOriginatorInfo struct {
	OriginatorCustomerNo   string `json:"originatorCustomerNo"`
	OriginatorCustomerName string `json:"originatorCustomerName"`
	OriginatorBankCode     string `json:"originatorBankCode"`
}
