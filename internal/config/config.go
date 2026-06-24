package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

// Config is the fully-parsed weather station configuration read from confdb.
type Config struct {
	APIKey       string
	Lat          float64
	Lon          float64
	PollInterval time.Duration
	MQTT         MQTTConfig
}

// MQTTConfig holds broker connection settings.
type MQTTConfig struct {
	Server string // scheme-prefixed: tcp://host or unix:///abs/path.sock
	Port   int    // used for tcp:// only
	Topic  string
}

// wire mirrors the confdb JSON document under the "weather" request key.
type wire struct {
	APIKey   string `json:"api-key"`
	Location struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	} `json:"location"`
	PollInterval int `json:"poll-interval"`
	MQTT         struct {
		Server string `json:"server"`
		Port   int    `json:"port"`
		Topic  string `json:"topic"`
	} `json:"mqtt"`
}

// Parse converts a confdb JSON document into a Config.
func Parse(data []byte) (Config, error) {
	if len(data) == 0 {
		return Config{}, errors.New("empty confdb document")
	}
	var w wire
	if err := json.Unmarshal(data, &w); err != nil {
		return Config{}, err
	}
	return Config{
		APIKey:       w.APIKey,
		Lat:          w.Location.Lat,
		Lon:          w.Location.Lon,
		PollInterval: time.Duration(w.PollInterval) * time.Second,
		MQTT: MQTTConfig{
			Server: w.MQTT.Server,
			Port:   w.MQTT.Port,
			Topic:  w.MQTT.Topic,
		},
	}, nil
}

// LoadFromConfdb reads the weather view from confdb via snapctl and parses it.
// snapctl is invoked with -d to emit a JSON document.
func LoadFromConfdb() (Config, error) {
	out, err := exec.Command(
		"snapctl", "get", "-d", ":weather-admin", "--view", "weather",
	).Output()
	if err != nil {
		return Config{}, fmt.Errorf("snapctl get weather view: %w", err)
	}
	return Parse(out)
}
