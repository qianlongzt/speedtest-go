package schema

import (
	"time"
)

type TelemetryData struct {
	Timestamp time.Time
	IPAddress string
	ISPInfo   string
	Extra     string
	UserAgent string
	Language  string
	Download  string
	Upload    string
	Ping      string
	Jitter    string
	Log       string
	UUID      string
}

type Config struct {
	File     string
	Hostname string
	Username string
	Password string
	Database string
}

type DataAccess interface {
	Insert(*TelemetryData) error
	FetchByUUID(string) (*TelemetryData, error)
	FetchLast100() ([]TelemetryData, error)
}
