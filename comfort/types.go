package comfort

// CreateOperationItem is one payout item for POST /v1/operations/create.
type CreateOperationItem struct {
	GUID                 *string    `json:"guid,omitempty"`
	Amount               string     `json:"amount"`
	Purpose              *string    `json:"purpose,omitempty"`
	PayoutPAN            *string    `json:"payout_pan,omitempty"`
	RefundOnFailedPayout *bool      `json:"refund_on_failed_payout,omitempty"`
	Recipient            *Recipient `json:"recipient,omitempty"`
}

type Recipient struct {
	LastName   string `json:"last_name"`
	FirstName  string `json:"first_name"`
	Patronymic string `json:"patronymic"`
	Phone      string `json:"phone"`

	DocumentType          *string `json:"document_type,omitempty"`
	DocumentNumber        *string `json:"document_number,omitempty"`
	DocumentSeries        *string `json:"document_series,omitempty"`
	DocumentIssuedCountry *string `json:"document_issued_country,omitempty"`
}

// CreateOperationsRequest is the payload for POST /v1/operations/create.
// Docs define this endpoint body as an object with RAW_BODY array.
type CreateOperationsRequest struct {
	RawBody []CreateOperationItem `json:"RAW_BODY,omitempty"`
}

type CreateOperationsResponseItem struct {
	GUID     string `json:"guid"`
	PublicID string `json:"public_id"`
}

// RefundOperationsRequest corresponds to POST /v1/operations/refund.
type RefundOperationsRequest struct {
	RawBody []string `json:"RAW_BODY"`
}

// OperationsStatusRequest corresponds to POST /v1/operations/status.
type OperationsStatusRequest struct {
	GUID *string `json:"guid,omitempty"`
}

type OperationsStatusResponse struct {
	Status   string `json:"status"`
	PublicID string `json:"public_id"`
}

// ChangeRecipientDataRequest corresponds to POST /v1/operations/change-recipient-data.
type ChangeRecipientDataRequest struct {
	GUID      string              `json:"guid"`
	Recipient ChangeRecipientData `json:"recipient"`
}

type ChangeRecipientData struct {
	LastName              string  `json:"last_name"`
	FirstName             string  `json:"first_name"`
	Patronymic            string  `json:"patronymic"`
	DocumentType          *string `json:"document_type,omitempty"`
	DocumentNumber        *string `json:"document_number,omitempty"`
	DocumentSeries        *string `json:"document_series,omitempty"`
	DocumentIssuedCountry *string `json:"document_issued_country,omitempty"`
}

type ExportFormat string

const (
	ExportFormatCSV  ExportFormat = "CSV"
	ExportFormatJSON ExportFormat = "JSON"
	ExportFormatXLSX ExportFormat = "XLSX"
)

// ExportOperationsRequest corresponds to POST /v1/export-operations.
type ExportOperationsRequest struct {
	FromDate       string        `json:"from_date"`
	ToDate         string        `json:"to_date"`
	Format         *ExportFormat `json:"format,omitempty"`
	RecepientEmail string        `json:"recepient_email"`
}

type ExportOperationsResponse struct {
	ExportID    string `json:"export_id"`
	Status      string `json:"status"`
	RequestedAt string `json:"requested_at"`
}

type BalanceResponse struct {
	Balance string `json:"balance"`
}
