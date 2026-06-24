package publisher

import "testing"

func TestBrokerURL(t *testing.T) {
	cases := []struct {
		name   string
		server string
		port   int
		want   string
	}{
		{"tcp appends port", "tcp://broker.local", 1883, "tcp://broker.local:1883"},
		{"unix absolute path untouched", "unix:///run/mosquitto/mqtt.sock", 0, "unix:///run/mosquitto/mqtt.sock"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := brokerURL(tc.server, tc.port)
			if err != nil {
				t.Fatalf("brokerURL error: %v", err)
			}
			if got != tc.want {
				t.Errorf("brokerURL(%q,%d)=%q want %q", tc.server, tc.port, got, tc.want)
			}
		})
	}
}

func TestBrokerURLRejectsUnknownScheme(t *testing.T) {
	if _, err := brokerURL("ws://broker", 0); err == nil {
		t.Error("brokerURL accepted ws scheme, want error")
	}
}
