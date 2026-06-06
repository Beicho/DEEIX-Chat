package security

import "errors"

var (
	ErrPoWStoreUnavailable          = errors.New("pow store unavailable")
	ErrPoWChallengeNotFound         = errors.New("pow challenge not found")
	ErrPoWChallengeExpired          = errors.New("pow challenge expired")
	ErrPoWChallengeReplay           = errors.New("pow challenge replay")
	ErrPoWActionMismatch            = errors.New("pow action mismatch")
	ErrPoWInvalidDifficulty         = errors.New("pow difficulty invalid")
	ErrPoWInvalidProof              = errors.New("pow proof invalid")
	ErrBrowserKeyNotFound           = errors.New("browser key not found")
	ErrBrowserKeyRevoked            = errors.New("browser key revoked")
	ErrBrowserKeyInvalid            = errors.New("browser key invalid")
	ErrRequestProofMissing          = errors.New("request proof missing")
	ErrRequestProofSession          = errors.New("request proof session mismatch")
	ErrRequestProofTimestamp        = errors.New("request proof timestamp invalid")
	ErrRequestProofBodyHashMismatch = errors.New("request proof body hash mismatch")
	ErrRequestProofSignature        = errors.New("request proof signature invalid")
	ErrRequestProofNonceReplay      = errors.New("request proof nonce replay")
)
