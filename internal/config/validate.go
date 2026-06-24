package config

import (
	"fmt"
	"strings"
	"time"
)

// MinPollInterval is the floor enforced to protect the OpenWeather quota.
const MinPollInterval = 60 * time.Second

// Validate checks that the configuration is complete and within bounds.
func (c Config) Validate() error {
	if strings.TrimSpace(c.APIKey) == "" {
		return fmt.Errorf("api-key must be set")
	}
	if c.Lat < -90 || c.Lat > 90 {
		return fmt.Errorf("lat must be between -90 and 90, got %v", c.Lat)
	}
	if c.Lon < -180 || c.Lon > 180 {
		return fmt.Errorf("lon must be between -180 and 180, got %v", c.Lon)
	}
	if c.PollInterval < MinPollInterval {
		return fmt.Errorf("poll-interval must be >= %s, got %s", MinPollInterval, c.PollInterval)
	}
	if strings.TrimSpace(c.MQTT.Topic) == "" {
		return fmt.Errorf("mqtt.topic must be set")
	}
	return c.MQTT.validateServer()
}

func (m MQTTConfig) validateServer() error {
	switch {
	case strings.HasPrefix(m.Server, "unix://"):
		return nil
	case strings.HasPrefix(m.Server, "tcp://"):
		if m.Port < 1 || m.Port > 65535 {
			return fmt.Errorf("mqtt.port must be 1..65535 for tcp, got %d", m.Port)
		}
		return nil
	default:
		return fmt.Errorf("mqtt.server must use a tcp:// or unix:// scheme, got %q", m.Server)
	}
}
