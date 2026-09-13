package transactionhistory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	snap "github.com/koriebruh/go-snap-bi"
)

// SourceOfFund describes one source of funds used for a transaction,
// per the standard's "Definisi Tipe" reference. Source is Mandatory;
// Amount's container is Optional (only its own value/currency members
// are Mandatory), so it's modeled as *snap.Money — corrected from an
// earlier version that had it as a plain (non-pointer) snap.Money,
// found by re-checking the dedicated SourceOfFund type-definition
// table directly rather than leaving it unverified.
type SourceOfFund struct {
	Source string      `json:"source"`
	Amount *snap.Money `json:"amount,omitempty"`
}

// TransactionDetail is one entry in a TransactionHistoryListResponse's
// detailData array. Amount's container is Optional per the Guides tab
// (only its own value/currency members are Mandatory) — corrected from
// an earlier version of this type that modeled it as an always-present
// plain snap.Money, against a direct portal re-verification recorded in
// docs/research/2026-09-13-registrasi-informasi-saldo-riwayat-transaksi-portal-research.md.
// This differs from the otherwise-similar-looking Transaction History
// Detail (Service Code 13) endpoint, where research marks the
// equivalent amount container Mandatory — recorded per-occurrence, not
// harmonized.
type TransactionDetail struct {
	DateTime       string          `json:"dateTime,omitempty"`
	Amount         *snap.Money     `json:"amount,omitempty"`
	Remark         string          `json:"remark,omitempty"`
	SourceOfFunds  []SourceOfFund  `json:"sourceOfFunds,omitempty"`
	Status         string          `json:"status"`
	Type           string          `json:"type"`
	AdditionalInfo json.RawMessage `json:"additionalInfo,omitempty"`
}

// TransactionHistoryListRequest is the request body for API Transaction
// History List (Service Code 12).
//
// PageSize and PageNumber are typed string, not int: the Guides tab types
// them "Integer", but the standard's own worked example sends them as
// quoted JSON strings (e.g. "10") — this models the actual wire example.
type TransactionHistoryListRequest struct {
	PartnerReferenceNo string          `json:"partnerReferenceNo,omitempty"`
	FromDateTime       string          `json:"fromDateTime,omitempty"`
	ToDateTime         string          `json:"toDateTime,omitempty"`
	PageSize           string          `json:"pageSize,omitempty"`
	PageNumber         string          `json:"pageNumber,omitempty"`
	AdditionalInfo     json.RawMessage `json:"additionalInfo,omitempty"`
}

// TransactionHistoryListResponse is the response body for API Transaction
// History List.
type TransactionHistoryListResponse struct {
	ResponseCode       string              `json:"responseCode"`
	ResponseMessage    string              `json:"responseMessage"`
	ReferenceNo        string              `json:"referenceNo,omitempty"`
	PartnerReferenceNo string              `json:"partnerReferenceNo,omitempty"`
	DetailData         []TransactionDetail `json:"detailData,omitempty"`
	AdditionalInfo     json.RawMessage     `json:"additionalInfo,omitempty"`
}

// TransactionHistoryList calls the SNAP Transaction History List endpoint
// (Service Code 12, path .../{version}/transaction-history-list). hb must
// already carry every field snap.HeaderBuilder needs except Body, which
// TransactionHistoryList sets itself so the exact marshaled bytes are used
// for both signing and the wire body.
func TransactionHistoryList(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req TransactionHistoryListRequest) (TransactionHistoryListResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return TransactionHistoryListResponse{}, fmt.Errorf("snap: transaction history list: encode request: %w", err)
	}
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return TransactionHistoryListResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return TransactionHistoryListResponse{}, fmt.Errorf("snap: transaction history list: %w", err)
	}

	var resp TransactionHistoryListResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return TransactionHistoryListResponse{}, fmt.Errorf("snap: transaction history list: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return TransactionHistoryListResponse{}, errors.New("snap: transaction history list: response has no responseCode")
	}
	return resp, nil
}
