package model

type ReportRow struct {
	SNo         int
	DCNoDate    string
	PartyName   string
	Items       string
	Rate        float64
	AdvanceRecd float64
	ToolHire    float64
	BalanceDue  float64
	Date        string
	Remarks     string
}
type DeliveryReportFilter struct {
	DateRange string // "7days", "14days", "21days", "1month", "2months", "1year", "2year"
	Remark    string // optional remark status
}
