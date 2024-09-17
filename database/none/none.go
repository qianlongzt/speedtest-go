package none

import (
	"github.com/librespeed/speedtest/database/schema"
)

type None struct{}

func Open(_ schema.Config) (schema.DataAccess, error) {
	return &None{}, nil
}

func (n *None) Insert(_ *schema.TelemetryData) error {
	return nil
}

func (n *None) FetchByUUID(_ string) (*schema.TelemetryData, error) {
	return &schema.TelemetryData{}, nil
}

func (n *None) FetchLast100() ([]schema.TelemetryData, error) {
	return []schema.TelemetryData{}, nil
}
