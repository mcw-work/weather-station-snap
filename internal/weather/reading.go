package weather

import (
	"encoding/json"
	"time"
)

// Reading is the normalized weather sample published to MQTT.
type Reading struct {
	TempC      float64
	Humidity   int
	Pressure   int
	WindSpeed  float64
	Conditions string
	Timestamp  time.Time
}

// payloadJSON controls field order and names of the published document.
type payloadJSON struct {
	TempC      float64 `json:"temp_c"`
	Humidity   int     `json:"humidity"`
	Pressure   int     `json:"pressure"`
	WindSpeed  float64 `json:"wind_speed"`
	Conditions string  `json:"conditions"`
	TS         string  `json:"ts"`
}

// Payload marshals the Reading into the compact published JSON document.
func (r Reading) Payload() ([]byte, error) {
	return json.Marshal(payloadJSON{
		TempC:      r.TempC,
		Humidity:   r.Humidity,
		Pressure:   r.Pressure,
		WindSpeed:  r.WindSpeed,
		Conditions: r.Conditions,
		TS:         r.Timestamp.UTC().Format(time.RFC3339),
	})
}
