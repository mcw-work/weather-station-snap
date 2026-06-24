package config

import (
	"testing"
	"time"
)

func TestParseValidJSON(t *testing.T) {
	in := []byte(`{
		"api-key": "abc123",
		"location": {"lat": 51.4545, "lon": -2.5879},
		"poll-interval": 600,
		"mqtt": {"server": "tcp://broker.local", "port": 1883, "topic": "weather/bristol"}
	}`)

	cfg, err := Parse(in)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cfg.APIKey != "abc123" {
		t.Errorf("APIKey = %q, want abc123", cfg.APIKey)
	}
	if cfg.Lat != 51.4545 || cfg.Lon != -2.5879 {
		t.Errorf("Lat/Lon = %v/%v, want 51.4545/-2.5879", cfg.Lat, cfg.Lon)
	}
	if cfg.PollInterval != 600*time.Second {
		t.Errorf("PollInterval = %v, want 10m", cfg.PollInterval)
	}
	if cfg.MQTT.Server != "tcp://broker.local" || cfg.MQTT.Port != 1883 || cfg.MQTT.Topic != "weather/bristol" {
		t.Errorf("MQTT = %+v, unexpected", cfg.MQTT)
	}
}

func TestParseEmptyIsError(t *testing.T) {
	if _, err := Parse([]byte("")); err == nil {
		t.Error("Parse(empty) returned nil error, want error")
	}
}
