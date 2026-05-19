package models

import "time"

type WabaPricing struct {
	ID              string    `json:"id"`
	WabaID          string    `json:"waba_id"`
	PhoneNumber     string    `json:"phone_number"`
	PricingCategory string    `json:"pricing_category"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	Volume          int       `json:"volume"`
	Cost            float64   `json:"cost"`
	CreatedAt       time.Time `json:"created_at"`
}

type PricingSummary struct {
	Month           string  `json:"month"`
	PricingCategory string  `json:"pricing_category"`
	TotalVolume     int     `json:"total_volume"`
	TotalCost       float64 `json:"total_cost"`
}

type PricingAnalyticsResponse struct {
	Data []PricingAnalyticsEntry `json:"data"`
}

type PricingAnalyticsEntry struct {
	Name       string             `json:"name"`
	DataPoints []PricingDataPoint `json:"data_points"`
}

type PricingDataPoint struct {
	Start           int64   `json:"start"`
	End             int64   `json:"end"`
	PhoneNumber     string  `json:"phone_number"`
	PricingCategory string  `json:"pricing_category"`
	Volume          int     `json:"volume"`
	Cost            float64 `json:"cost"`
}
