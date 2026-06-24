package config

import (
	"strings"
	"testing"
	"time"
)

func validConfig() Config {
	return Config{
		APIKey:       "abc123",
		Lat:          51.4545,
		Lon:          -2.5879,
		PollInterval: 600 * time.Second,
		MQTT:         MQTTConfig{Server: "tcp://broker.local", Port: 1883, Topic: "weather/bristol"},
	}
}

func TestValidateOK(t *testing.T) {
	if err := validConfig().Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
}

func TestValidateUnixServerOK(t *testing.T) {
	c := validConfig()
	c.MQTT.Server = "unix:///run/mosquitto/mqtt.sock"
	c.MQTT.Port = 0 // port irrelevant for unix
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate() unix = %v, want nil", err)
	}
}

func TestValidateRejects(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{"missing api-key", func(c *Config) { c.APIKey = "" }, "api-key"},
		{"lat too high", func(c *Config) { c.Lat = 91 }, "lat"},
		{"lon too low", func(c *Config) { c.Lon = -181 }, "lon"},
		{"poll below floor", func(c *Config) { c.PollInterval = 30 * time.Second }, "poll-interval"},
		{"bad scheme", func(c *Config) { c.MQTT.Server = "broker.local" }, "scheme"},
		{"empty topic", func(c *Config) { c.MQTT.Topic = "" }, "topic"},
		{"tcp missing port", func(c *Config) { c.MQTT.Port = 0 }, "port"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := validConfig()
			tc.mutate(&c)
			err := c.Validate()
			if err == nil {
				t.Fatalf("Validate() = nil, want error containing %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("Validate() = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}
