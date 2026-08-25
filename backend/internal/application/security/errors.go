package security

import (
	"errors"

	domainsecurity "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/security"
)

var (
	ErrPoWStoreUnavailable          = domainsecurity.ErrPoWStoreUnavailable
	ErrPoWChallengeNotFound         = domainsecurity.ErrPoWChallengeNotFound
	ErrPoWChallengeExpired          = errors.New("pow challenge expired")
	ErrPoWChallengeReplay           = errors.New("pow challenge replay")
	ErrPoWActionMismatch            = errors.New("pow action mismatch")
	ErrPoWInvalidDifficulty         = errors.New("pow difficulty invalid")
	ErrPoWInvalidProof              = errors.New("pow proof invalid")
	ErrBrowserKeyNotFound           = domainsecurity.ErrBrowserKeyNotFound
	ErrBrowserKeyRevoked            = domainsecurity.ErrBrowserKeyRevoked
	ErrBrowserKeyInvalid            = errors.New("browser key invalid")
	ErrRequestProofMissing          = errors.New("request proof missing")
	ErrRequestProofSession          = errors.New("request proof session mismatch")
	ErrRequestProofTimestamp        = errors.New("request proof timestamp invalid")
	ErrRequestProofBodyHashMismatch = errors.New("request proof body hash mismatch")
	ErrRequestProofSignature        = errors.New("request proof signature invalid")
	ErrRequestProofNonceReplay      = errors.New("request proof nonce replay")
)
