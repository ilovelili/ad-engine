package domain

import "time"

const (
	PlatformInstagram = "instagram"
)

type PlatformConnection struct {
	ID                   uint   `gorm:"primaryKey"`
	Platform             string `gorm:"index:idx_platform_connections_platform_account,unique"`
	AccountLabel         string
	AccountIdentifier    string `gorm:"index:idx_platform_connections_platform_account,unique"`
	ExternalAccountID    string
	Status               string
	CredentialNonce      string
	CredentialCiphertext string
	Scopes               string
	LastValidatedAt      *time.Time
	LastSyncAt           *time.Time
	LastError            string
	MetadataJSON         string `gorm:"type:text"`
	AdAccountsJSON       string `gorm:"type:text"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type AdAccount struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	Currency string `json:"currency"`
	Timezone string `json:"timezone"`
}

type PlatformConnectionMetadata struct {
	DisplayName                string `json:"displayName,omitempty"`
	InstagramBusinessAccountID string `json:"instagramBusinessAccountId,omitempty"`
}

type PlatformConnectionSnapshot struct {
	ID                         uint        `json:"id"`
	Platform                   string      `json:"platform"`
	AccountLabel               string      `json:"accountLabel"`
	AccountIdentifier          string      `json:"accountIdentifier"`
	ExternalAccountID          string      `json:"externalAccountId"`
	Status                     string      `json:"status"`
	DisplayName                string      `json:"displayName"`
	InstagramBusinessAccountID string      `json:"instagramBusinessAccountId,omitempty"`
	Scopes                     []string    `json:"scopes"`
	LastValidatedAt            *time.Time  `json:"lastValidatedAt,omitempty"`
	LastSyncAt                 *time.Time  `json:"lastSyncAt,omitempty"`
	LastError                  string      `json:"lastError,omitempty"`
	AdAccounts                 []AdAccount `json:"adAccounts"`
}

type SupportedPlatform struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	AuthenticationModel string   `json:"authenticationModel"`
	Fields              []string `json:"fields"`
	Notes               []string `json:"notes"`
}

type PlatformConnectionsView struct {
	SupportedPlatforms []SupportedPlatform          `json:"supportedPlatforms"`
	Connections        []PlatformConnectionSnapshot `json:"connections"`
}
