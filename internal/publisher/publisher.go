package publisher

import (
	"fmt"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Publisher wraps an MQTT client connection.
type Publisher struct {
	client mqtt.Client
	topic  string
}

// brokerURL converts our scheme-prefixed server + port into a paho broker URL.
// tcp://host        -> tcp://host:port
// unix:///abs/path  -> unchanged (paho dials the unix socket path)
func brokerURL(server string, port int) (string, error) {
	switch {
	case strings.HasPrefix(server, "unix://"):
		return server, nil
	case strings.HasPrefix(server, "tcp://"):
		return fmt.Sprintf("%s:%d", server, port), nil
	default:
		return "", fmt.Errorf("unsupported mqtt server scheme: %q", server)
	}
}

// New connects to the broker described by server/port and returns a Publisher.
// clientID identifies this snap to the broker.
func New(server string, port int, topic, clientID string) (*Publisher, error) {
	broker, err := brokerURL(server, port)
	if err != nil {
		return nil, err
	}
	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID).
		SetConnectTimeout(15 * time.Second).
		SetAutoReconnect(true)

	client := mqtt.NewClient(opts)
	tok := client.Connect()
	if !tok.WaitTimeout(15 * time.Second) {
		return nil, fmt.Errorf("timed out connecting to %s", broker)
	}
	if err := tok.Error(); err != nil {
		return nil, fmt.Errorf("connecting to %s: %w", broker, err)
	}
	return &Publisher{client: client, topic: topic}, nil
}

// Publish sends a payload to the configured topic at QoS 0, retained.
func (p *Publisher) Publish(payload []byte) error {
	tok := p.client.Publish(p.topic, 0, true, payload)
	if !tok.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("timed out publishing to %s", p.topic)
	}
	return tok.Error()
}

// Close disconnects from the broker.
func (p *Publisher) Close() {
	p.client.Disconnect(250)
}
