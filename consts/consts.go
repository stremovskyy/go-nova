package consts

const (
	HeaderXSign       = "x-sign"
	HeaderXMerchantID = "x-merchant-id"
	HeaderAccept      = "Accept"
	HeaderContentType = "Content-Type"

	ContentTypeJSON = "application/json"
)

// Base URLs.
const (
	// External API (acquiring/checkout).
	DefaultAcquiringBaseURL = "https://api-qecom.novapay.ua" // test
	ProductionAcquiringURL  = "https://api-ecom.novapay.ua"  // prod

	// Comfort API.
	DefaultComfortBaseURL = "https://contragent-api.novapay.ua"
)

// Acquiring (External API) endpoint paths.
const (
	AcquiringCreateSessionPath       = "/v1/session"
	AcquiringAddPaymentPath          = "/v1/payment"
	AcquiringVoidSessionPath         = "/v1/void"
	AcquiringCompleteHoldPath        = "/v1/complete-hold"
	AcquiringExpireSessionPath       = "/v1/expire"
	AcquiringConfirmDeliveryPath     = "/v1/confirm-delivery-hold"
	AcquiringPrintExpressWaybillPath = "/v1/print-express-waybill"
	AcquiringGetStatusPath           = "/v1/get-status"
	AcquiringDeliveryPricePath       = "/v1/delivery-price"
)

// Checkout (External API) endpoint paths.
const (
	CheckoutCreateSessionPath = "/v1/checkout/session"
	CheckoutAddPaymentPath    = "/v1/checkout/payment"
	CheckoutVoidSessionPath   = "/v1/void"
	CheckoutGetStatusPath     = "/v1/get-status"
	CheckoutExpireSessionPath = "/v1/expire"
)

// Comfort API endpoint paths.
const (
	ComfortCreateOperationsPath    = "/v1/operations/create"
	ComfortRefundOperationsPath    = "/v1/operations/refund"
	ComfortOperationsStatusPath    = "/v1/operations/status"
	ComfortChangeRecipientDataPath = "/v1/operations/change-recipient-data"
	ComfortBalancePath             = "/v1/balance"
	ComfortExportOperationsPath    = "/v1/export-operations"
)
