package consts

// SessionStatus is the status of an acquiring payment session.
//
// Values are taken from the NovaPay documentation.
type SessionStatus string

const (
	SessionStatusCreated                  SessionStatus = "created"
	SessionStatusExpired                  SessionStatus = "expired"
	SessionStatusProcessing               SessionStatus = "processing"
	SessionStatusHolded                   SessionStatus = "holded"
	SessionStatusHoldConfirmed            SessionStatus = "hold_confirmed"
	SessionStatusProcessingHoldCompletion SessionStatus = "processing_hold_completion"
	SessionStatusPaid                     SessionStatus = "paid"
	SessionStatusFailed                   SessionStatus = "failed"
	SessionStatusProcessingVoid           SessionStatus = "processing_void"
	SessionStatusVoided                   SessionStatus = "voided"
)
