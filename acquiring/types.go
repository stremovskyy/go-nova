package acquiring

import "encoding/json"

// CreateSessionRequest corresponds to "Create session" (POST /v1/session).
type CreateSessionRequest struct {
	MerchantID string `json:"merchant_id"`

	ClientFirstName  *string `json:"client_first_name,omitempty"`
	ClientLastName   *string `json:"client_last_name,omitempty"`
	ClientPatronymic *string `json:"client_patronymic,omitempty"`
	ClientPhone      string  `json:"client_phone"`
	ClientEmail      *string `json:"client_email,omitempty"`

	CallbackURL *string `json:"callback_url,omitempty"`
	SuccessURL  *string `json:"success_url,omitempty"`
	FailURL     *string `json:"fail_url,omitempty"`

	SuccessRedirectTimeout *int32          `json:"success_redirect_timeout,omitempty"`
	Metadata               json.RawMessage `json:"metadata,omitempty"`
}

type CreateSessionResponse struct {
	ID       string          `json:"id"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// AddPaymentRequest corresponds to "Add payment" (POST /v1/payment).
type AddPaymentRequest struct {
	MerchantID string  `json:"merchant_id"`
	SessionID  string  `json:"session_id"`
	Amount     float64 `json:"amount"`
	ExternalID *string `json:"external_id,omitempty"`

	UseHold    *bool     `json:"use_hold,omitempty"`
	Identifier *string   `json:"identifier,omitempty"`
	Delivery   *Delivery `json:"delivery,omitempty"`
	Products   []Product `json:"products,omitempty"`
}

type Delivery struct {
	VolumeWeight       float64 `json:"volume_weight"`
	Weight             float64 `json:"weight"`
	RecipientCity      string  `json:"recipient_city"`
	RecipientWarehouse string  `json:"recipient_warehouse"`
}

type Product struct {
	Description string  `json:"description"`
	Count       int32   `json:"count"`
	Price       float64 `json:"price"`
}

type AddPaymentResponse struct {
	ID            string   `json:"id"`
	URL           string   `json:"url"`
	DeliveryPrice *float64 `json:"delivery_price,omitempty"`
}

// SessionRequest is the payload used by endpoints that require merchant_id + session_id.
type SessionRequest struct {
	MerchantID string `json:"merchant_id"`
	SessionID  string `json:"session_id"`
}

// CompleteHoldRequest corresponds to "Complete hold" (POST /v1/complete-hold).
type CompleteHoldRequest struct {
	MerchantID string                  `json:"merchant_id"`
	SessionID  string                  `json:"session_id"`
	Amount     *float64                `json:"amount,omitempty"`
	Operations []CompleteHoldOperation `json:"operations,omitempty"`
}

type CompleteHoldOperation struct {
	ID                  string  `json:"id"`
	Amount              float64 `json:"amount"`
	RecipientIdentifier string  `json:"recipient_identifier"`
}

type ConfirmDeliveryHoldResponse struct {
	ID             string          `json:"id"`
	ExpressWaybill string          `json:"express_waybill"`
	RefID          string          `json:"ref_id"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
}

// DeliveryPriceRequest corresponds to "Delivery price" (POST /v1/delivery-price).
type DeliveryPriceRequest struct {
	MerchantID         string  `json:"merchant_id"`
	RecipientCity      string  `json:"recipient_city"`
	RecipientWarehouse string  `json:"recipient_warehouse"`
	VolumeWeight       float64 `json:"volume_weight"`
	Weight             float64 `json:"weight"`
	Amount             float64 `json:"amount"`
}

// DeliveryPriceResponse schema is not fully described in public docs; keep it generic.
type DeliveryPriceResponse map[string]any

// GetStatusResponse corresponds to "Get status" (POST /v1/get-status).
type GetStatusResponse struct {
	ID               string          `json:"id"`
	Metadata         json.RawMessage `json:"metadata,omitempty"`
	Paytype          string          `json:"paytype"`
	ApprovalCode     *string         `json:"approval_code,omitempty"`
	TerminalName     *string         `json:"terminal_name,omitempty"`
	Status           string          `json:"status"`
	CreatedAt        string          `json:"created_at"`
	ClientPhone      *string         `json:"client_phone,omitempty"`
	ClientFirstName  *string         `json:"client_first_name,omitempty"`
	ClientLastName   *string         `json:"client_last_name,omitempty"`
	ClientPatronymic *string         `json:"client_patronymic,omitempty"`
	Pan              *string         `json:"pan,omitempty"`
	Operations       []OperationInfo `json:"operations,omitempty"`
}

type OperationInfo struct {
	ExternalID *string `json:"external_id,omitempty"`
	Amount     float64 `json:"amount"`
}

// Postback is the current v3 callback payload from NovaPay.
type Postback struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	Paytype      string `json:"paytype"`
	TerminalName string `json:"terminal_name"`
	RRN          string `json:"RRN"`
	APPROVAL     int64  `json:"APPROVAL"`

	CreatedAt string          `json:"created_at"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`

	ClientFirstName  string  `json:"client_first_name"`
	ClientLastName   string  `json:"client_last_name"`
	ClientPatronymic *string `json:"client_patronymic,omitempty"`
	ClientPhone      string  `json:"client_phone"`
	ClientEmail      *string `json:"client_email,omitempty"`
	ClientIP         *string `json:"client_ip,omitempty"`

	ProcessingResult string            `json:"processing_result"`
	CardDetails      *PostbackCard     `json:"card_details,omitempty"`
	Payments         []PostbackPayment `json:"payments,omitempty"`
}

type PostbackCard struct {
	Pan         string `json:"pan"`
	CardBank    string `json:"card_bank"`
	CardCountry string `json:"card_country"`
	CardType    string `json:"card_type"`
}

type PostbackPayment struct {
	ExternalID *string   `json:"external_id,omitempty"`
	Amount     float64   `json:"amount"`
	Products   []Product `json:"products,omitempty"`
}
