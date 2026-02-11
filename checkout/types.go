package checkout

// CreateSessionRequest corresponds to "Create checkout session" (POST /v1/checkout/session).
type CreateSessionRequest struct {
	MerchantID           string           `json:"merchant_id"`
	CallbackURL          string           `json:"callback_url"`
	SuccessURL           *string          `json:"success_url,omitempty"`
	FailURL              *string          `json:"fail_url,omitempty"`
	ClientPhone          *string          `json:"client_phone,omitempty"`
	CreateExpressWaybill *bool            `json:"create_express_waybill,omitempty"`
	Delivery             *SessionDelivery `json:"delivery,omitempty"`
}

type SessionDelivery struct {
	VolumeWeight float64 `json:"volume_weight"`
	Weight       float64 `json:"weight"`
}

// AddPaymentRequest corresponds to "Add checkout payment" (POST /v1/checkout/payment).
type AddPaymentRequest struct {
	MerchantID string    `json:"merchant_id"`
	SessionID  string    `json:"session_id"`
	ExternalID *string   `json:"external_id,omitempty"`
	UseHold    *bool     `json:"use_hold,omitempty"`
	Identifier *string   `json:"identifier,omitempty"`
	Amount     float64   `json:"amount"`
	Products   []Product `json:"products,omitempty"`
}

type Product struct {
	Description *string `json:"description,omitempty"`
	Count       int32   `json:"count"`
	Price       float64 `json:"price"`
	Image       *string `json:"image,omitempty"`
}

// SessionRequest is used by checkout endpoints that require merchant_id + session_id.
type SessionRequest struct {
	MerchantID string `json:"merchant_id"`
	SessionID  string `json:"session_id"`
}

// GenericResponse is used where docs do not fully define response schema.
type GenericResponse map[string]any
