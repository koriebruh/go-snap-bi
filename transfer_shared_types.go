package snap

import "encoding/json"

// TransferOriginatorInfo is the shared "originatorInfos[]" entry used by
// the Trigger Transfer sub-group. All three fields are String per the
// Guides tab.
type TransferOriginatorInfo struct {
	OriginatorCustomerNo   string `json:"originatorCustomerNo"`
	OriginatorCustomerName string `json:"originatorCustomerName"`
	OriginatorBankCode     string `json:"originatorBankCode"`
}

// InterbankBulkTransferItem is one entry in Interbank Bulk Transfer's
// request "bulkObject[]" array.
type InterbankBulkTransferItem struct {
	PartnerReferenceNo     string                   `json:"partnerReferenceNo"`
	BankCode               string                   `json:"bankCode"`
	BeneficiaryAccountNo   string                   `json:"beneficiaryAccountNo"`
	BeneficiaryAccountName string                   `json:"beneficiaryAccountName"`
	Amount                 Money                    `json:"amount"`
	OriginatorInfos        []TransferOriginatorInfo `json:"originatorInfos,omitempty"`
	AdditionalInfo         json.RawMessage          `json:"additionalInfo,omitempty"`
}

// InterbankBulkTransferNotificationItem is one entry in Interbank Bulk
// Transfer - Notification's request "bulkObject[]" array — a
// settlement-result callback shape, distinct from
// InterbankBulkTransferItem's transfer-instruction shape.
type InterbankBulkTransferNotificationItem struct {
	PartnerReferenceNo string `json:"partnerReferenceNo"`
	ResponseCode       string `json:"responseCode"`
	ResponseMessage    string `json:"responseMessage"`
}
