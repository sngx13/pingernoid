package models

import (
	"github.com/google/uuid"
)

type MeasurementResultAlerts struct {
	MsrID          uuid.UUID `json:"msr_id" gorm:"type:uuid"`
	AlertTimestamp string    `json:"alert_timestamp"`
	AlertReason    string    `json:"alert_reason"`
	AlertMessage   string    `json:"alert_message"`
}

type MeasurementResults struct {
	MsrID        uuid.UUID `json:"msr_id" gorm:"type:uuid"`
	Timestamp    string    `json:"timestamp"`
	Rcvd         int       `json:"rcvd"`
	Sent         int       `json:"sent"`
	Loss         float64   `json:"loss"`
	AvgRtt       float64   `json:"avg_rtt"`
	MinRtt       float64   `json:"min_rtt"`
	MaxRtt       float64   `json:"max_rtt"`
	Jitter       float64   `json:"jitter"`
	IPHopCount   int       `json:"ip_hop_count"`
	ASHopCount   int       `json:"as_hop_count"`
	IPPath       string    `json:"ip_path"`
	ASPath       string    `json:"as_path"`
	CombinedPath string    `json:"combined_path"`
	Alerting     bool      `json:"alerting"`
}

type PingMeasurement struct {
	ID          uuid.UUID                 `json:"id" gorm:"primary_key;type:uuid"`
	CreatedAt   string                    `json:"created_at"`
	LastPollAt  string                    `json:"last_poll_at"`
	StoppedAt   string                    `json:"stopped_at"`
	Target      string                    `json:"target" gorm:"unique"`
	PacketCount int                       `json:"packet_count"`
	IsHostname  bool                      `json:"is_hostname"`
	Frequency   int                       `json:"frequency"`
	Results     []MeasurementResults      `json:"results" gorm:"foreignkey:MsrID;constraint:OnDelete:CASCADE"`
	Alerts      []MeasurementResultAlerts `json:"alerts" gorm:"foreignkey:MsrID;constraint:OnDelete:CASCADE"`
	Status      int                       `json:"status"`
	StatusName  string                    `json:"status_name"`
}

type SiteVisitor struct {
	IPAddress   string `json:"ip_address" gorm:"primary_key"`
	ISP         string `json:"isp"`
	ASN         string `json:"asn"`
	Country     string `json:"country"`
	CountryCode string `json:"country_code"`
}
