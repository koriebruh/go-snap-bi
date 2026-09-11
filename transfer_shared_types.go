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

// LocalizedText is the shared {english, indonesia} bilingual text shape
// used across the Virtual Account sub-group (e.g. inquiryReason,
// paymentFlagReason, billDescription, per-bill reason, freeTexts[]
// entries). Both fields are String, unmarked for M/O in the source, so
// both carry omitempty.
type LocalizedText struct {
	English   string `json:"english,omitempty"`
	Indonesia string `json:"indonesia,omitempty"`
}

// BillDetail is one entry in the Virtual Account sub-group's
// "billDetails[]" array (max 24 entries). All fields are unmarked for
// M/O in the source, so all carry omitempty; BillDescription, BillAmount,
// and Reason are pointers since omitempty has no effect on a
// non-pointer struct value. BillReferenceNo is documented Numeric in
// the Guides tab (research §3); per the package's ambiguous-type rule,
// a documented Numeric field is typed json.RawMessage rather than
// string, since a plain string field fails the entire decode if any
// issuer sends it as a bare JSON number, matching the existing
// AccountTransactionLimit/APIKey precedent.
type BillDetail struct {
	BillCode        string          `json:"billCode,omitempty"`
	BillNo          string          `json:"billNo,omitempty"`
	BillName        string          `json:"billName,omitempty"`
	BillShortName   string          `json:"billShortName,omitempty"`
	BillDescription *LocalizedText  `json:"billDescription,omitempty"`
	BillSubCompany  string          `json:"billSubCompany,omitempty"`
	BillAmount      *Money          `json:"billAmount,omitempty"`
	AdditionalInfo  json.RawMessage `json:"additionalInfo,omitempty"`
	BillAmountLabel string          `json:"billAmountLabel,omitempty"`
	BillAmountValue string          `json:"billAmountValue,omitempty"`
	BillReferenceNo json.RawMessage `json:"billReferenceNo,omitempty"`
	Status          string          `json:"status,omitempty"`
	Reason          *LocalizedText  `json:"reason,omitempty"`
}
