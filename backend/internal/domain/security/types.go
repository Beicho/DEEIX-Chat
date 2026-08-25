package security

import (
	"errors"
	"time"
)

var (
	ErrPoWStoreUnavailable  = errors.New("pow store unavailable")
	ErrPoWChallengeNotFound = errors.New("pow challenge not found")
	ErrBrowserKeyNotFound   = errors.New("browser key not found")
	ErrBrowserKeyRevoked    = errors.New("browser key revoked")
)

// StoredPoWChallenge is the persistence representation of an issued challenge.
type StoredPoWChallenge struct {
	Challenge  string
	UserID     uint
	Action     string
	Difficulty int
	ExpiresAt  time.Time
}

// PoWChallenge is a short-lived proof-of-work challenge issued by the server.
type PoWChallenge struct {
	Challenge  string
	Difficulty int
	Action     string
	ExpiresAt  time.Time
}

// PoWProof is the client solution for a PoWChallenge.
type PoWProof struct {
	Challenge  string
	Nonce      string
	Hash       string
	Difficulty int
}

// RequestProof is sent by the browser for protected high-cost endpoints.
type RequestProof struct {
	KeyID     string
	SessionID string
	Timestamp int64
	Nonce     string
	BodyHash  string
	Signature string
	PoWProof  *PoWProof
}

// BrowserKey is the public key bound to a user session.
type BrowserKey struct {
	UserID       uint
	SessionID    string
	KeyID        string
	PublicKeyJWK string
	CreatedAt    time.Time
	LastUsedAt   time.Time
	RevokedAt    *time.Time
}

// DeviceFingerprint stores stable browser and hardware signals for risk analysis.
type DeviceFingerprint struct {
	ID                  uint
	UserID              uint
	FingerprintID       string
	ScreenResolution    string
	ColorDepth          int
	PixelRatio          float64
	HardwareConcurrency int
	DeviceMemory        int
	MaxTouchPoints      int
	UserAgent           string
	Language            string
	Timezone            string
	Platform            string
	CanvasHash          string
	WebGLVendor         string
	WebGLRenderer       string
	FontsHash           string
	AudioHash           string
	IPAddress           string
	TLSFingerprint      string
	FirstSeenAt         time.Time
	LastSeenAt          time.Time
	SeenCount           int
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// FingerprintAssociation links one fingerprint to multiple accounts.
type FingerprintAssociation struct {
	ID              uint
	FingerprintID   string
	UserIDs         []uint
	ConfidenceScore float64
	RiskLevel       string
	DetectedAt      time.Time
	IgnoredAt       *time.Time
	Reason          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// MultiAccountDetection describes the risk evaluation for a shared fingerprint.
type MultiAccountDetection struct {
	FingerprintID   string
	AssociatedUsers []uint
	ConfidenceScore float64
	RiskLevel       string
}
