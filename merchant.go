package go_nova

// Merchant holds NovaPay merchant information.
//
// Today the API requires passing merchant_id inside request bodies.
// This struct is a convenience to keep integrations tidy.
type Merchant struct {
	ID string
}

func NewMerchant(id string) Merchant {
	return Merchant{ID: id}
}
