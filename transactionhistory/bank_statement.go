package transactionhistory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// BankStatementBalanceAmount is the {value, currency, dateTime} shape used
// by BankStatementBalance's StartingBalance and EndingBalance members. It
// is distinct from snap.Money because it carries a dateTime member Money
// doesn't have; all three members are Mandatory per the field table.
type BankStatementBalanceAmount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
	DateTime string `json:"dateTime"`
}

// BankStatementBalance is one entry in BankStatementResponse's balance
// array: the starting and ending balance before the first/last
// transaction. Amount, StartingBalance, and EndingBalance all carry a
// dateTime member snap.Money lacks (per the worked example), so all three
// use the richer BankStatementBalanceAmount shape, not plain snap.Money.
// All three members are Mandatory per the field table.
type BankStatementBalance struct {
	Amount          BankStatementBalanceAmount `json:"amount"`
	StartingBalance BankStatementBalanceAmount `json:"startingBalance"`
	EndingBalance   BankStatementBalanceAmount `json:"endingBalance"`
}

// BankStatementEntryTotal is the shape shared by BankStatementResponse's
// totalCreditEntries and totalDebitEntries fields (identical per the field
// table, so they share this one type).
//
// NumberOfEntries is typed string, not int: the field table types it
// "int", but the standard's own worked example sends it as a quoted JSON
// string (e.g. "10") — this models the actual wire example, matching the
// PageSize/PageNumber precedent in transaction_history.go.
type BankStatementEntryTotal struct {
	NumberOfEntries string     `json:"numberOfEntries,omitempty"`
	Amount          snap.Money `json:"amount"`
}

// BankStatementDetailBalanceEntry is one entry in a
// BankStatementDetailBalance's StartAmount/EndAmount arrays.
type BankStatementDetailBalanceEntry struct {
	Amount snap.Money `json:"amount"`
}

// BankStatementDetailBalance is a BankStatementDetail's detailBalance
// object: the balance immediately before (StartAmount) and after
// (EndAmount) that one transaction.
type BankStatementDetailBalance struct {
	StartAmount []BankStatementDetailBalanceEntry `json:"startAmount,omitempty"`
	EndAmount   []BankStatementDetailBalanceEntry `json:"endAmount,omitempty"`
}

// BankStatementDetail is one entry in BankStatementResponse's detailData
// array: a single transaction line.
type BankStatementDetail struct {
	DetailBalance           *BankStatementDetailBalance `json:"detailBalance,omitempty"`
	Amount                  snap.Money                  `json:"amount"`
	OriginAmount            snap.Money                  `json:"originAmount"`
	TransactionDate         string                      `json:"transactionDate"`
	Remark                  string                      `json:"remark"`
	TransactionID           string                      `json:"transactionId,omitempty"`
	Type                    string                      `json:"type"`
	TransactionDetailStatus string                      `json:"transactionDetailStatus,omitempty"`
	DetailInfo              json.RawMessage             `json:"detailInfo,omitempty"`
}

// BankStatementRequest is the request body for API Bank Statement (Service
// Code 14, Riwayat Transaksi group).
//
// BankCardToken and AccountNo are Conditional and mutually exclusive: per
// the spec, BankCardToken must be filled if AccountNo is null and vice
// versa. This package validates wire shape only, never business rules, so
// both are just tagged omitempty; enforcing the XOR is left to the caller.
type BankStatementRequest struct {
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	BankCardToken      string          `json:"bankCardToken,omitempty"`
	AccountNo          string          `json:"accountNo,omitempty"`
	FromDateTime       string          `json:"fromDateTime,omitempty"`
	ToDateTime         string          `json:"toDateTime,omitempty"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// BankStatementResponse is the response body for API Bank Statement.
type BankStatementResponse struct {
	ResponseCode       string                   `json:"responseCode"`
	ResponseMessage    string                   `json:"responseMessage"`
	ReferenceNo        string                   `json:"referenceNo,omitempty"`
	PartnerReferenceNo string                   `json:"partnerReferenceNo,omitempty"`
	Balance            []BankStatementBalance   `json:"balance,omitempty"`
	TotalCreditEntries *BankStatementEntryTotal `json:"totalCreditEntries,omitempty"`
	TotalDebitEntries  *BankStatementEntryTotal `json:"totalDebitEntries,omitempty"`
	HasMore            string                   `json:"hasMore,omitempty"`
	LastRecordDateTime string                   `json:"lastRecordDateTime,omitempty"`
	DetailData         []BankStatementDetail    `json:"detailData,omitempty"`
	AdditionalInfo     json.RawMessage          `json:"additionalInfo,omitempty"`
}

// BankStatement calls the SNAP API Bank Statement endpoint (Service Code
// 14, path .../{version}/bank-statement). hb must already carry every
// field snap.HeaderBuilder needs except Body, which BankStatement sets
// itself so the exact marshaled bytes are used for both signing and the
// wire body.
func BankStatement(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req BankStatementRequest) (BankStatementResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return BankStatementResponse{}, fmt.Errorf("snap: bank statement: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return BankStatementResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return BankStatementResponse{}, fmt.Errorf("snap: bank statement: %w", err)
	}

	var resp BankStatementResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return BankStatementResponse{}, fmt.Errorf("snap: bank statement: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return BankStatementResponse{}, errors.New("snap: bank statement: response has no responseCode")
	}
	return resp, nil
}
