package snap

// Money is the shared {value, currency} amount shape used across every
// per-service request or response that carries a monetary value. value
// is a decimal string (e.g. "200000.00"); currency is ISO 4217 (e.g.
// "IDR").
//
// Extracted into its own file (previously defined in balance_inquiry.go,
// now in the balanceinfo subpackage) because it is a core cross-cutting
// type shared by every subpackage, not specific to BalanceInquiry.
type Money struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}
