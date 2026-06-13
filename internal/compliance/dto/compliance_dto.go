package dto

type ComplianceCheckResponse struct {
	Passed   bool    `json:"passed"`
	Reason   string  `json:"reason,omitempty"`
	LimitUSD float64 `json:"limit_usd"`
}
