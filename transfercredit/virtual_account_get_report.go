package transfercredit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	snap "github.com/koriebruh/go-snap-bi"
)

// VAGetReportRequest is the request body for API VA - Get Report
// (Service Code 35). PartnerServiceID is the only mandatory field per
// the Guides tab; research §5.3 line 155 documents its type as
// flipping to Number for this endpoint specifically (String everywhere
// else in the package), so it is json.RawMessage — see the Phase 15
// design doc's ambiguous-type table. This endpoint is documented both
// as GET (Guides tab) and POST-with-body (code snippet); this package
// models it as POST with a JSON body, matching every other endpoint in
// the group — see the design doc for the unresolved GET/POST
// contradiction (research §6 item 1).
type VAGetReportRequest struct {
	PartnerServiceID json.RawMessage `json:"partnerServiceId"`
	StartDate        string          `json:"startDate,omitempty"`
	StartTime        string          `json:"startTime,omitempty"`
	EndDate          string          `json:"endDate,omitempty"`
	EndTime          string          `json:"endTime,omitempty"`
	AdditionalInfo   json.RawMessage `json:"additionalInfo,omitempty"`
}

// GetReportData is one entry in VAGetReportResponse's "virtualAccountdata"
// array (lowercase d, research §5.3 line 132) — field-identical to
// VAInquiryStatusData, per §5.3 line 155's "each item shaped like the
// Payment/Inquiry-Status response object." It is a distinct type (not
// a reuse), per the package's "distinct types per service code even
// for identical shapes" convention; transfer_shared_types_test.go
// guards the two against field/tag drift with a full-equality check.
type GetReportData struct {
	PartnerServiceID        string          `json:"partnerServiceId,omitempty"`
	CustomerNo              json.RawMessage `json:"customerNo,omitempty"`
	VirtualAccountNo        string          `json:"virtualAccountNo,omitempty"`
	TrxID                   string          `json:"trxId,omitempty"`
	PaymentRequestID        string          `json:"paymentRequestId,omitempty"`
	ChannelCode             json.RawMessage `json:"channelCode,omitempty"`
	HashedSourceAccountNo   string          `json:"hashedSourceAccountNo,omitempty"`
	SourceBankCode          string          `json:"sourceBankCode,omitempty"`
	PaidAmount              *snap.Money     `json:"paidAmount,omitempty"`
	CumulativePaymentAmount *snap.Money     `json:"cumulativePaymentAmount,omitempty"`
	PaidBills               string          `json:"paidBills,omitempty"`
	TotalAmount             *snap.Money     `json:"totalAmount,omitempty"`
	TrxDateTime             string          `json:"trxDateTime,omitempty"`
	ReferenceNo             string          `json:"referenceNo,omitempty"`
	JournalNum              string          `json:"journalNum,omitempty"`
	PaymentType             json.RawMessage `json:"paymentType,omitempty"`
	FlagAdvise              string          `json:"flagAdvise,omitempty"`
	SubCompany              string          `json:"subCompany,omitempty"`
	BillDetails             []BillDetail    `json:"billDetails,omitempty"`
	FreeTexts               []LocalizedText `json:"freeTexts,omitempty"`
	PaymentFlagReason       *LocalizedText  `json:"paymentFlagReason,omitempty"`
	PaymentFlagStatus       string          `json:"paymentFlagStatus,omitempty"`
	AdditionalInfo          json.RawMessage `json:"additionalInfo,omitempty"`
	TransactionDate         string          `json:"transactionDate,omitempty"`
}

// VAGetReportResponse is the response body for API VA - Get Report. It
// is the only VA response in the package whose data field is an array
// rather than a single object (research §5.3 line 155).
type VAGetReportResponse struct {
	ResponseCode       string          `json:"responseCode"`
	ResponseMessage    string          `json:"responseMessage"`
	VirtualAccountData []GetReportData `json:"virtualAccountdata,omitempty"`
}

// VAGetReport calls the SNAP VA - Get Report endpoint (Service Code
// 35, path .../{version}/transfer-va/report — the path was previously
// misstated here as transfer-va/get-report with no recorded
// justification, corrected against research §1's own path table). hb
// must already carry every field snap.HeaderBuilder needs except
// Method and Body:
// VAGetReport sets Method to POST itself — this is the one VA endpoint
// where the source spec contradicts itself (GET per the Guides tab,
// POST-with-body per the code snippet), and the package has chosen
// POST (see the type comment above), so the method is not
// caller-configurable, the same way the PUT/DELETE endpoints in Phase
// 13 fix their own method — and Body, so the exact marshaled bytes are
// used for both signing and the wire body.
//
// A report covering a wide date range can return an unbounded number
// of array entries; the shared transport layer caps every response
// body at 10 MiB (see transport.go), so callers pulling large reports
// should page by narrower date/time ranges rather than one unbounded
// call.
func VAGetReport(ctx context.Context, t *snap.Transport, hb snap.HeaderBuilder, req VAGetReportRequest) (VAGetReportResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return VAGetReportResponse{}, fmt.Errorf("snap: va get report: encode request: %w", err)
	}
	hb.Method = http.MethodPost
	hb.Body = body

	env, err := t.Do(ctx, hb)
	if err != nil {
		return VAGetReportResponse{}, err
	}
	if err := snap.CheckResponseStatus(env.ResponseCode, env.StatusCode); err != nil {
		return VAGetReportResponse{}, fmt.Errorf("snap: va get report: %w", err)
	}

	var resp VAGetReportResponse
	if err := json.Unmarshal(env.Raw, &resp); err != nil {
		return VAGetReportResponse{}, fmt.Errorf("snap: va get report: decode response: %w", err)
	}
	if resp.ResponseCode == "" {
		return VAGetReportResponse{}, errors.New("snap: va get report: response has no responseCode")
	}
	return resp, nil
}
