package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/canonical/weather-station-snap/internal/config"
	"github.com/canonical/weather-station-snap/internal/publisher"
	"github.com/canonical/weather-station-snap/internal/weather"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("weatherstationd: ")

	cfg, err := config.LoadFromConfdb()
	if err != nil {
		log.Fatalf("cannot load config from confdb: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	clientID := "weatherstation-" + hostnameOr("snap")
	pub, err := publisher.New(cfg.MQTT.Server, cfg.MQTT.Port, cfg.MQTT.Topic, clientID)
	if err != nil {
		log.Fatalf("cannot connect to MQTT broker: %v", err)
	}
	defer pub.Close()

	wc := weather.NewClient(cfg.APIKey)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	pollAndPublish(ctx, wc, pub, cfg) // run once immediately
	for {
		select {
		case <-ctx.Done():
			log.Print("shutting down")
			return
		case <-ticker.C:
			pollAndPublish(ctx, wc, pub, cfg)
		}
	}
}

func pollAndPublish(ctx context.Context, wc *weather.Client, pub *publisher.Publisher, cfg config.Config) {
	reading, err := wc.Fetch(ctx, cfg.Lat, cfg.Lon)
	if err != nil {
		log.Printf("fetch failed: %v", err)
		return
	}
	payload, err := reading.Payload()
	if err != nil {
		log.Printf("payload marshaling failed: %v", err)
		return
	}
	if err := pub.Publish(payload); err != nil {
		log.Printf("publish failed: %v", err)
		return
	}
	log.Printf("published reading to %s", cfg.MQTT.Topic)
}

func hostnameOr(fallback string) string {
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return fallback
}
