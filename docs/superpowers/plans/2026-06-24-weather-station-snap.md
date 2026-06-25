# Weather Station Snap Implementation Plan

> **For agentic workers:** REQUIRED: Use the `subagent-driven-development` agent (recommended) or `executing-plans` agent to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a strictly-confined Core26 snap that polls weather data from the OpenWeather API on a configurable interval and publishes a normalized JSON payload to a configurable MQTT topic, over either a TCP or unix-domain-socket broker connection, configured entirely through snapd confdb.

**Architecture:** A single Go daemon (`weatherstationd`, `daemon: simple`) acts as its own confdb **custodian**. A Python `configure` hook validates `snap set` values and writes the complete config into confdb; a Python `change-view` hook validates every write before commit. The daemon reads the committed config back from confdb via `snapctl get`, then runs a poll → transform → publish loop. The configure hook restarts the daemon after a successful write so it picks up new config; an `observe-view` hook restarts it when an external snap writes the view. MQTT transport (TCP vs unix socket) is selected by a scheme-prefixed `server` config value (`tcp://host` / `unix:///path`), handled natively by the paho Go MQTT client.

**Tech Stack:**
- **Language:** Go (1.22+)
- **MQTT:** `github.com/eclipse/paho.mqtt.golang` (native `tcp://`, `ssl://`, and `unix://` scheme support)
- **HTTP/JSON:** Go standard library (`net/http`, `encoding/json`)
- **Weather API:** OpenWeather Current Weather Data (`/data/2.5/weather`, `units=metric`)
- **Packaging:** snapcraft, `base: core26`, strict confinement
- **Config:** snapd confdb (custodian/observer model); hooks in Python 3 and POSIX shell
- **Testing:** Go `testing` + `net/http/httptest`; Python `unittest`; `snapcraft` + local `mosquitto` for integration

---

## Configuration Surface (single source of truth)

These names are used consistently across the schema, hooks, daemon, and tests. Do not rename them between tasks.

**snap config keys** (what an operator runs `snap set weatherstation <key>=<value>` on):

| Key | Type | Notes |
|-----|------|-------|
| `api-key` | string | OpenWeather API key. Secret. |
| `lat` | number | Latitude, −90..90 |
| `lon` | number | Longitude, −180..180 |
| `poll-interval` | int | Seconds between polls, minimum 60, default 600 |
| `mqtt.server` | string | Scheme-prefixed: `tcp://host` or `unix:///abs/path.sock` |
| `mqtt.port` | int | 1..65535. Used for `tcp://` only; ignored for `unix://` |
| `mqtt.topic` | string | Topic to publish to |

**confdb storage tree** (`v1` version prefix for evolution):

```
v1.weather.api-key            (string, visibility: secret)
v1.weather.location.lat       (number, min -90, max 90)
v1.weather.location.lon       (number, min -180, max 180)
v1.weather.poll-interval      (int, min 60)
v1.weather.mqtt.server        (string)
v1.weather.mqtt.port          (int, min 1, max 65535)
v1.weather.mqtt.topic         (string)
```

**confdb views** (schema name is `weather`, so view names are kept bare per the naming convention for a tightly-scoped schema; plug names remain `weather-admin`/`weather-state` = `<schema>-<view-name>`):
- `admin` → `v1.weather`, access `read-write` (custodian; daemon reads the secret api-key here via the `weather-admin` plug)
- `state` → `v1.weather`, access `read` (read-only view for future observer snaps; api-key redacted)

**Published MQTT payload** (normalized, compact JSON, single configured topic):

```json
{"temp_c":18.4,"humidity":72,"pressure":1012,"wind_speed":3.2,"conditions":"Clouds","ts":"2026-06-24T10:00:00Z"}
```

> **Placeholder to resolve before signing:** `<ACCOUNT_ID>` is the Canonical/store account ID used in the confdb plug definitions and the schema assertion `account-id`. Replace every `<ACCOUNT_ID>` occurrence with the real value before signing/importing the assertion.

---

## File Structure

```
weather-station-snap/
├── go.mod
├── go.sum
├── cmd/
│   └── weatherstationd/
│       └── main.go                 # wiring: load config, ticker loop, signals, shutdown
├── internal/
│   ├── config/
│   │   ├── config.go               # Config/MQTTConfig types, LoadFromConfdb, Parse
│   │   ├── config_test.go
│   │   ├── validate.go             # Validate(): lat/lon, server scheme, poll floor, topic
│   │   └── validate_test.go
│   ├── weather/
│   │   ├── client.go               # OpenWeather client + normalization to Reading
│   │   ├── client_test.go
│   │   ├── reading.go              # Reading struct + JSON payload marshaling
│   │   └── reading_test.go
│   └── publisher/
│       ├── publisher.go            # broker URL build (tcp/unix), connect, Publish
│       └── publisher_test.go
├── schema/
│   └── weather-confdb-schema.yaml  # confdb-schema assertion body (storage + views)
├── snap/
│   ├── snapcraft.yaml
│   ├── hooks/
│   │   ├── configure               # Python: validate snap config, write confdb, restart daemon
│   │   ├── change-view-weather-admin   # Python: validate writes before commit
│   │   ├── observe-view-weather-admin  # shell: restart daemon on external write
│   │   └── connect-plug-weather-admin  # shell: log connect, restart daemon
│   └── local/
│       └── configuration/
│           └── defaults/
│               └── config.yaml     # placeholder defaults (shape + unset detection)
├── tests/
│   └── hooks/
│       ├── test_configure.py
│       └── test_change_view.py
└── README.md
```

**Responsibilities (one job per file):**
- `internal/config` — owns the config shape, confdb read/parse, and validation. No network, no MQTT.
- `internal/weather` — owns OpenWeather access and normalization. No confdb, no MQTT.
- `internal/publisher` — owns MQTT transport selection and publishing. No weather, no confdb.
- `cmd/weatherstationd/main.go` — wires the three together with a ticker and graceful shutdown.
- Hooks — own all confdb writes/validation; the daemon only reads.

---

## Task 0: Repository and Go module scaffolding

**Files:**
- Create: `go.mod`
- Create: `README.md`
- Create: `.gitignore`

- [ ] **Step 1: Initialize the repository and Go module**

Run:
```bash
mkdir -p weather-station-snap && cd weather-station-snap
git init
go mod init github.com/<ACCOUNT_ID>/weather-station-snap
```
Expected: `go.mod` created with `module github.com/<ACCOUNT_ID>/weather-station-snap` and a `go 1.22` line.

- [ ] **Step 2: Add the MQTT dependency**

Run:
```bash
go get github.com/eclipse/paho.mqtt.golang@latest
```
Expected: `go.mod`/`go.sum` updated; `require github.com/eclipse/paho.mqtt.golang vX.Y.Z` present.

- [ ] **Step 3: Create `.gitignore`**

```gitignore
/parts
/prime
/stage
*.snap
/weatherstationd
```

- [ ] **Step 4: Create a minimal `README.md`**

```markdown
# weather-station-snap

A Core26 snap that polls the OpenWeather API and publishes normalized weather
readings to an MQTT topic (TCP or unix socket), configured via snapd confdb.

## Configure

    snap set weatherstation api-key=YOUR_KEY
    snap set weatherstation lat=51.4545 lon=-2.5879
    snap set weatherstation poll-interval=600
    snap set weatherstation mqtt.server=tcp://broker.local mqtt.port=1883
    snap set weatherstation mqtt.topic=weather/bristol

For a unix-socket broker:

    snap set weatherstation mqtt.server=unix:///run/mosquitto/mqtt.sock
    snap set weatherstation mqtt.topic=weather/bristol
```

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum README.md .gitignore
git commit -m "chore: scaffold repo, Go module, and MQTT dependency"
```

---

## Task 1: Config types and confdb parsing

**Files:**
- Create: `internal/config/config.go`
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing test**

`internal/config/config_test.go`:
```go
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
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/config/ -run TestParse -v`
Expected: FAIL — `undefined: Parse` / package does not compile.

- [ ] **Step 3: Write the minimal implementation**

`internal/config/config.go`:
```go
package config

import (
	"encoding/json"
	"errors"
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
		return Config{}, err
	}
	return Parse(out)
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/config/ -run TestParse -v`
Expected: PASS (both `TestParseValidJSON` and `TestParseEmptyIsError`).

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): parse confdb weather document into Config"
```

---

## Task 2: Config validation

**Files:**
- Create: `internal/config/validate.go`
- Test: `internal/config/validate_test.go`

- [ ] **Step 1: Write the failing test**

`internal/config/validate_test.go`:
```go
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
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/config/ -run TestValidate -v`
Expected: FAIL — `c.Validate undefined`.

- [ ] **Step 3: Write the minimal implementation**

`internal/config/validate.go`:
```go
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
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/config/ -v`
Expected: PASS (all config tests).

- [ ] **Step 5: Commit**

```bash
git add internal/config/validate.go internal/config/validate_test.go
git commit -m "feat(config): validate completeness, bounds, and server scheme"
```

---

## Task 3: Weather Reading type and payload marshaling

**Files:**
- Create: `internal/weather/reading.go`
- Test: `internal/weather/reading_test.go`

- [ ] **Step 1: Write the failing test**

`internal/weather/reading_test.go`:
```go
package weather

import (
	"testing"
	"time"
)

func TestReadingPayload(t *testing.T) {
	r := Reading{
		TempC:      18.4,
		Humidity:   72,
		Pressure:   1012,
		WindSpeed:  3.2,
		Conditions: "Clouds",
		Timestamp:  time.Date(2026, 6, 24, 10, 0, 0, 0, time.UTC),
	}
	got, err := r.Payload()
	if err != nil {
		t.Fatalf("Payload() error: %v", err)
	}
	want := `{"temp_c":18.4,"humidity":72,"pressure":1012,"wind_speed":3.2,"conditions":"Clouds","ts":"2026-06-24T10:00:00Z"}`
	if string(got) != want {
		t.Errorf("Payload()=\n%s\nwant\n%s", got, want)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/weather/ -run TestReadingPayload -v`
Expected: FAIL — `undefined: Reading`.

- [ ] **Step 3: Write the minimal implementation**

`internal/weather/reading.go`:
```go
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
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/weather/ -run TestReadingPayload -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/weather/reading.go internal/weather/reading_test.go
git commit -m "feat(weather): Reading type with normalized JSON payload"
```

---

## Task 4: OpenWeather client (fetch + normalize)

**Files:**
- Create: `internal/weather/client.go`
- Test: `internal/weather/client_test.go`

- [ ] **Step 1: Write the failing test**

`internal/weather/client_test.go`:
```go
package weather

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const sampleResponse = `{
  "weather": [{"main": "Clouds", "description": "broken clouds"}],
  "main": {"temp": 18.4, "humidity": 72, "pressure": 1012},
  "wind": {"speed": 3.2},
  "dt": 1782640800
}`

func TestFetchNormalizes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("lat") != "51.4545" || q.Get("lon") != "-2.5879" {
			t.Errorf("lat/lon query = %s/%s", q.Get("lat"), q.Get("lon"))
		}
		if q.Get("units") != "metric" {
			t.Errorf("units = %q, want metric", q.Get("units"))
		}
		if q.Get("appid") != "abc123" {
			t.Errorf("appid = %q, want abc123", q.Get("appid"))
		}
		w.Write([]byte(sampleResponse))
	}))
	defer srv.Close()

	c := NewClient("abc123")
	c.BaseURL = srv.URL

	r, err := c.Fetch(context.Background(), 51.4545, -2.5879)
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if r.TempC != 18.4 || r.Humidity != 72 || r.Pressure != 1012 {
		t.Errorf("main fields = %+v", r)
	}
	if r.WindSpeed != 3.2 || r.Conditions != "Clouds" {
		t.Errorf("wind/conditions = %+v", r)
	}
	if !r.Timestamp.Equal(time.Unix(1782640800, 0).UTC()) {
		t.Errorf("timestamp = %v", r.Timestamp)
	}
}

func TestFetchHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient("bad")
	c.BaseURL = srv.URL
	_, err := c.Fetch(context.Background(), 0, 0)
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("Fetch() err = %v, want error containing 401", err)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/weather/ -run TestFetch -v`
Expected: FAIL — `undefined: NewClient`.

- [ ] **Step 3: Write the minimal implementation**

`internal/weather/client.go`:
```go
package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// DefaultBaseURL is the OpenWeather current-weather endpoint.
const DefaultBaseURL = "https://api.openweathermap.org/data/2.5/weather"

// Client fetches and normalizes OpenWeather current-weather data.
type Client struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

// NewClient returns a Client with sane defaults.
func NewClient(apiKey string) *Client {
	return &Client{
		APIKey:  apiKey,
		BaseURL: DefaultBaseURL,
		HTTP:    &http.Client{Timeout: 15 * time.Second},
	}
}

type apiResponse struct {
	Weather []struct {
		Main string `json:"main"`
	} `json:"weather"`
	Main struct {
		Temp     float64 `json:"temp"`
		Humidity int     `json:"humidity"`
		Pressure int     `json:"pressure"`
	} `json:"main"`
	Wind struct {
		Speed float64 `json:"speed"`
	} `json:"wind"`
	DT int64 `json:"dt"`
}

// Fetch retrieves current weather for the given coordinates and normalizes it.
func (c *Client) Fetch(ctx context.Context, lat, lon float64) (Reading, error) {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return Reading{}, err
	}
	q := u.Query()
	q.Set("lat", strconv.FormatFloat(lat, 'f', -1, 64))
	q.Set("lon", strconv.FormatFloat(lon, 'f', -1, 64))
	q.Set("units", "metric")
	q.Set("appid", c.APIKey)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Reading{}, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return Reading{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Reading{}, fmt.Errorf("openweather returned status %d", resp.StatusCode)
	}

	var ar apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return Reading{}, err
	}

	conditions := ""
	if len(ar.Weather) > 0 {
		conditions = ar.Weather[0].Main
	}
	return Reading{
		TempC:      ar.Main.Temp,
		Humidity:   ar.Main.Humidity,
		Pressure:   ar.Main.Pressure,
		WindSpeed:  ar.Wind.Speed,
		Conditions: conditions,
		Timestamp:  time.Unix(ar.DT, 0).UTC(),
	}, nil
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/weather/ -v`
Expected: PASS (all weather tests).

- [ ] **Step 5: Commit**

```bash
git add internal/weather/client.go internal/weather/client_test.go
git commit -m "feat(weather): OpenWeather client with normalization"
```

---

## Task 5: MQTT publisher (transport selection + publish)

**Files:**
- Create: `internal/publisher/publisher.go`
- Test: `internal/publisher/publisher_test.go`

> The `unix://` scheme is handled natively by `paho.mqtt.golang`: for an absolute path it dials `uri.Path`, so `unix:///run/mosquitto/mqtt.sock` connects to that socket. `tcp://host` requires the port appended as `host:port`. The publisher builds the broker URL accordingly; this task unit-tests the URL builder (pure function) without a live broker.

- [ ] **Step 1: Write the failing test**

`internal/publisher/publisher_test.go`:
```go
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
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/publisher/ -run TestBrokerURL -v`
Expected: FAIL — `undefined: brokerURL`.

- [ ] **Step 3: Write the minimal implementation**

`internal/publisher/publisher.go`:
```go
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
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/publisher/ -v`
Expected: PASS (`TestBrokerURL`, `TestBrokerURLRejectsUnknownScheme`).

- [ ] **Step 5: Commit**

```bash
git add internal/publisher/publisher.go internal/publisher/publisher_test.go
git commit -m "feat(publisher): MQTT publisher with tcp/unix transport selection"
```

---

## Task 6: Daemon wiring (main)

**Files:**
- Create: `cmd/weatherstationd/main.go`

> No unit test for `main` directly; the logic lives in the tested packages. This task wires them and is validated by `go build` and `go vet`. The poll loop runs once immediately, then on each tick.

- [ ] **Step 1: Write `cmd/weatherstationd/main.go`**

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/<ACCOUNT_ID>/weather-station-snap/internal/config"
	"github.com/<ACCOUNT_ID>/weather-station-snap/internal/publisher"
	"github.com/<ACCOUNT_ID>/weather-station-snap/internal/weather"
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
```

- [ ] **Step 2: Build and vet the whole module**

Run: `go build ./... && go vet ./...`
Expected: no output (success), binary compiles.

- [ ] **Step 3: Run the full test suite**

Run: `go test ./...`
Expected: PASS for `internal/config`, `internal/weather`, `internal/publisher`; `cmd/...` and root report `no test files` (OK).

- [ ] **Step 4: Commit**

```bash
git add cmd/weatherstationd/main.go
git commit -m "feat(daemon): wire config, weather, and publisher into poll loop"
```

---

## Task 7: confdb schema assertion body

**Files:**
- Create: `schema/weather-confdb-schema.yaml`

> This is the assertion **body** (storage + views). It is signed and imported separately (see Task 13). `api-key` is `visibility: secret` so it is redacted in the read-only `state` view; the daemon reads it through the read-write `admin` view (the `weather-admin` plug).

- [ ] **Step 1: Write `schema/weather-confdb-schema.yaml`**

```yaml
storage:
  schema:
    v1:
      type: map
      schema:
        weather:
          type: map
          schema:
            api-key:
              type: string
              visibility: secret
            location:
              type: map
              schema:
                lat:
                  type: number
                  min: -90
                  max: 90
                lon:
                  type: number
                  min: -180
                  max: 180
            poll-interval:
              type: int
              min: 60
            mqtt:
              type: map
              schema:
                server:
                  type: string
                port:
                  type: int
                  min: 1
                  max: 65535
                topic:
                  type: string

views:
  admin:
    rules:
      - request: weather
        storage: v1.weather
        access: read-write
  state:
    rules:
      - request: weather
        storage: v1.weather
        access: read
```

- [ ] **Step 2: Validate the YAML parses**

Run: `python3 -c "import yaml,sys; yaml.safe_load(open('schema/weather-confdb-schema.yaml'))" && echo OK`
Expected: `OK`.

- [ ] **Step 3: Commit**

```bash
git add schema/weather-confdb-schema.yaml
git commit -m "feat(confdb): weather schema storage and admin/state views"
```

---

## Task 8: snapcraft.yaml

**Files:**
- Create: `snap/snapcraft.yaml`

> Strict confinement. `network` plug covers OpenWeather HTTPS and TCP MQTT. The unix-socket case requires access to the broker's socket; this is provided via a `content` interface plug (`mqtt-socket`) that a broker snap shares into `$SNAP_DATA/mqtt`. When using a unix socket, set `mqtt.server=unix://$SNAP_DATA/mqtt/<socket-name>` (an absolute path resolved at runtime). Document this as a deployment requirement in the README during Task 12.

- [ ] **Step 1: Write `snap/snapcraft.yaml`**

```yaml
name: weatherstation
base: core26
version: "0.1.0"
summary: Publishes OpenWeather data to an MQTT topic
description: |
  Polls the OpenWeather current-weather API on a configurable interval and
  publishes a normalized JSON reading to a configurable MQTT topic over either
  a TCP or unix-domain-socket broker connection. Configured via snapd confdb.

confinement: strict
grade: devel

apps:
  weatherstationd:
    command: bin/weatherstationd
    daemon: simple
    restart-condition: on-failure
    plugs:
      - network
      - mqtt-socket

parts:
  weatherstationd:
    plugin: go
    source: .
    build-snaps:
      - go/1.22/stable
    organize:
      bin/weatherstationd: bin/weatherstationd
  hooks:
    plugin: dump
    source: snap/local
    organize:
      configuration: etc/configuration

plugs:
  weather-admin:
    interface: confdb
    account: <ACCOUNT_ID>
    view: weather/admin
    role: custodian

  # Read-only view, available for future observer snaps.
  weather-state:
    interface: confdb
    account: <ACCOUNT_ID>
    view: weather/state

  # Shares a broker's unix socket directory for the unix:// transport.
  mqtt-socket:
    interface: content
    content: mqtt-socket
    target: $SNAP_DATA/mqtt

hooks:
  configure:
    plugs: [weather-admin]
  change-view-weather-admin:
    plugs: [weather-admin]
  observe-view-weather-admin:
    plugs: [weather-admin]
  connect-plug-weather-admin:
    plugs: [weather-admin]
```

- [ ] **Step 2: Validate the YAML parses**

Run: `python3 -c "import yaml; yaml.safe_load(open('snap/snapcraft.yaml'))" && echo OK`
Expected: `OK`.

- [ ] **Step 3: Commit**

```bash
git add snap/snapcraft.yaml
git commit -m "feat(snap): snapcraft.yaml with daemon, confdb plugs, and hooks"
```

---

## Task 9: configure hook + tests

**Files:**
- Create: `snap/hooks/configure`
- Create: `snap/local/configuration/defaults/config.yaml`
- Test: `tests/hooks/test_configure.py`

> The configure hook reads `snap set` values, validates completeness (no partial writes), assembles the confdb JSON document, writes it via `snapctl set :weather-admin --view weather=<json>`, then restarts the daemon. It logs to syslog and always exits 0 (a failing hook would block snap operations). To make the write logic testable, the document-assembly is a pure function `build_document` importable by the test.

- [ ] **Step 1: Write the defaults file**

`snap/local/configuration/defaults/config.yaml`:
```yaml
api-key: "api-key-placeholder"
lat: 0
lon: 0
poll-interval: 600
mqtt-server: "mqtt-server-placeholder"
mqtt-port: 1883
mqtt-topic: "mqtt-topic-placeholder"
```

- [ ] **Step 2: Write the failing test**

`tests/hooks/test_configure.py`:
```python
import importlib.util
import os
import unittest

HOOK = os.path.join(os.path.dirname(__file__), "..", "..", "snap", "hooks", "configure")


def load_hook():
    spec = importlib.util.spec_from_file_location("configure_hook", HOOK)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


class TestBuildDocument(unittest.TestCase):
    def test_complete_values_produce_document(self):
        mod = load_hook()
        values = {
            "api-key": "abc123",
            "lat": "51.4545",
            "lon": "-2.5879",
            "poll-interval": "600",
            "mqtt.server": "tcp://broker.local",
            "mqtt.port": "1883",
            "mqtt.topic": "weather/bristol",
        }
        doc = mod.build_document(values)
        self.assertEqual(doc["api-key"], "abc123")
        self.assertEqual(doc["location"], {"lat": 51.4545, "lon": -2.5879})
        self.assertEqual(doc["poll-interval"], 600)
        self.assertEqual(
            doc["mqtt"],
            {"server": "tcp://broker.local", "port": 1883, "topic": "weather/bristol"},
        )

    def test_missing_value_raises(self):
        mod = load_hook()
        values = {"api-key": "abc123"}  # everything else missing
        with self.assertRaises(mod.IncompleteConfig):
            mod.build_document(values)

    def test_unix_server_allows_default_port(self):
        mod = load_hook()
        values = {
            "api-key": "abc123",
            "lat": "51.0",
            "lon": "-2.0",
            "poll-interval": "600",
            "mqtt.server": "unix:///run/mosquitto/mqtt.sock",
            "mqtt.port": "1883",
            "mqtt.topic": "weather/bristol",
        }
        doc = mod.build_document(values)
        self.assertTrue(doc["mqtt"]["server"].startswith("unix://"))


if __name__ == "__main__":
    unittest.main()
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `python3 -m unittest tests.hooks.test_configure -v`
Expected: FAIL — cannot import hook / `build_document` undefined.

- [ ] **Step 4: Write the configure hook**

`snap/hooks/configure`:
```python
#!/usr/bin/env python3
import json
import logging
import logging.handlers
import subprocess
import sys

logger = logging.getLogger("configure-hook")
logger.setLevel(logging.INFO)
try:
    _h = logging.handlers.SysLogHandler(address="/dev/log")
    _h.setFormatter(logging.Formatter("weatherstation.configure: %(message)s"))
    logger.addHandler(_h)
except Exception:
    pass

REQUIRED = [
    "api-key",
    "lat",
    "lon",
    "poll-interval",
    "mqtt.server",
    "mqtt.port",
    "mqtt.topic",
]


class IncompleteConfig(Exception):
    """Raised when one or more required snap config values are unset."""


def build_document(values):
    """Assemble the confdb JSON document from snap config values.

    Raises IncompleteConfig if any required key is missing/empty.
    """
    missing = [k for k in REQUIRED if not values.get(k)]
    if missing:
        raise IncompleteConfig("missing: " + ", ".join(missing))
    return {
        "api-key": values["api-key"],
        "location": {
            "lat": float(values["lat"]),
            "lon": float(values["lon"]),
        },
        "poll-interval": int(values["poll-interval"]),
        "mqtt": {
            "server": values["mqtt.server"],
            "port": int(values["mqtt.port"]),
            "topic": values["mqtt.topic"],
        },
    }


def get(key):
    r = subprocess.run(
        ["snapctl", "get", key], capture_output=True, text=True, check=False
    )
    if r.returncode == 0 and r.stdout.strip():
        return r.stdout.strip()
    return None


def main():
    values = {k: get(k) for k in REQUIRED}
    try:
        doc = build_document(values)
    except IncompleteConfig as e:
        logger.info("Configuration incomplete (%s); skipping confdb write", e)
        return

    result = subprocess.run(
        ["snapctl", "set", ":weather-admin", "--view", "weather=" + json.dumps(doc)],
        capture_output=True,
        text=True,
        check=False,
    )
    if result.returncode != 0:
        logger.warning("Failed to write confdb: %s", result.stderr.strip())
        return
    logger.info("Configuration written to confdb")

    restart = subprocess.run(
        ["snapctl", "restart", "weatherstation.weatherstationd"],
        capture_output=True,
        text=True,
        check=False,
    )
    if restart.returncode != 0:
        logger.info("Daemon restart skipped/failed: %s", restart.stderr.strip())


if __name__ == "__main__":
    try:
        main()
    except Exception as e:  # never fail the hook
        logger.error("configure hook error: %s", e)
    sys.exit(0)
```

- [ ] **Step 5: Make the hook executable**

Run: `chmod +x snap/hooks/configure`
Expected: file mode includes execute bit.

- [ ] **Step 6: Run the test to verify it passes**

Run: `python3 -m unittest tests.hooks.test_configure -v`
Expected: PASS (3 tests).

- [ ] **Step 7: Commit**

```bash
git add snap/hooks/configure snap/local/configuration/defaults/config.yaml tests/hooks/test_configure.py
git commit -m "feat(hooks): configure hook writes validated confdb document"
```

---

## Task 10: change-view hook + tests

**Files:**
- Create: `snap/hooks/change-view-weather-admin`
- Test: `tests/hooks/test_change_view.py`

> The custodian's change-view hook validates every write to `weather-admin` before commit, including writes from other snaps. It exits non-zero to abort an invalid transaction. Validation logic is a pure function `validate(doc)` returning an error string or `None`, so it is unit-testable without snapctl.

- [ ] **Step 1: Write the failing test**

`tests/hooks/test_change_view.py`:
```python
import importlib.util
import os
import unittest

HOOK = os.path.join(
    os.path.dirname(__file__), "..", "..", "snap", "hooks", "change-view-weather-admin"
)


def load_hook():
    spec = importlib.util.spec_from_file_location("change_view_hook", HOOK)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def valid_doc():
    return {
        "api-key": "abc123",
        "location": {"lat": 51.45, "lon": -2.59},
        "poll-interval": 600,
        "mqtt": {"server": "tcp://broker.local", "port": 1883, "topic": "weather/bristol"},
    }


class TestValidate(unittest.TestCase):
    def test_valid_returns_none(self):
        mod = load_hook()
        self.assertIsNone(mod.validate(valid_doc()))

    def test_unix_server_ok(self):
        mod = load_hook()
        d = valid_doc()
        d["mqtt"]["server"] = "unix:///run/mosquitto/mqtt.sock"
        self.assertIsNone(mod.validate(d))

    def test_bad_lat(self):
        mod = load_hook()
        d = valid_doc()
        d["location"]["lat"] = 200
        self.assertIn("lat", mod.validate(d))

    def test_poll_below_floor(self):
        mod = load_hook()
        d = valid_doc()
        d["poll-interval"] = 10
        self.assertIn("poll-interval", mod.validate(d))

    def test_bad_scheme(self):
        mod = load_hook()
        d = valid_doc()
        d["mqtt"]["server"] = "broker.local"
        self.assertIn("scheme", mod.validate(d))

    def test_tcp_bad_port(self):
        mod = load_hook()
        d = valid_doc()
        d["mqtt"]["port"] = 0
        self.assertIn("port", mod.validate(d))


if __name__ == "__main__":
    unittest.main()
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `python3 -m unittest tests.hooks.test_change_view -v`
Expected: FAIL — cannot import hook / `validate` undefined.

- [ ] **Step 3: Write the change-view hook**

`snap/hooks/change-view-weather-admin`:
```python
#!/usr/bin/env python3
import json
import logging
import logging.handlers
import subprocess
import sys

logger = logging.getLogger("change-view-hook")
logger.setLevel(logging.INFO)
try:
    _h = logging.handlers.SysLogHandler(address="/dev/log")
    _h.setFormatter(logging.Formatter("weatherstation.change-view: %(message)s"))
    logger.addHandler(_h)
except Exception:
    pass

MIN_POLL_INTERVAL = 60


def validate(doc):
    """Return an error string if doc is invalid, else None."""
    if not str(doc.get("api-key", "")).strip():
        return "api-key must be set"

    loc = doc.get("location", {})
    lat, lon = loc.get("lat"), loc.get("lon")
    if lat is None or not (-90 <= lat <= 90):
        return f"lat must be between -90 and 90, got {lat}"
    if lon is None or not (-180 <= lon <= 180):
        return f"lon must be between -180 and 180, got {lon}"

    poll = doc.get("poll-interval")
    if poll is None or poll < MIN_POLL_INTERVAL:
        return f"poll-interval must be >= {MIN_POLL_INTERVAL}, got {poll}"

    mqtt = doc.get("mqtt", {})
    if not str(mqtt.get("topic", "")).strip():
        return "mqtt.topic must be set"

    server = str(mqtt.get("server", ""))
    if server.startswith("unix://"):
        return None
    if server.startswith("tcp://"):
        port = mqtt.get("port")
        if port is None or not (1 <= port <= 65535):
            return f"mqtt.port must be 1..65535 for tcp, got {port}"
        return None
    return "mqtt.server must use a tcp:// or unix:// scheme"


def read_incoming():
    out = subprocess.run(
        ["snapctl", "get", "-d", ":weather-admin", "--view", "weather"],
        capture_output=True,
        text=True,
        check=True,
    ).stdout.strip()
    if not out:
        return {}
    return json.loads(out)


def main():
    doc = read_incoming()
    err = validate(doc)
    if err:
        logger.error("rejecting confdb write: %s", err)
        sys.exit(1)  # abort the transaction
    logger.info("confdb write validated")
    sys.exit(0)


if __name__ == "__main__":
    main()
```

- [ ] **Step 4: Make the hook executable**

Run: `chmod +x snap/hooks/change-view-weather-admin`
Expected: execute bit set.

- [ ] **Step 5: Run the test to verify it passes**

Run: `python3 -m unittest tests.hooks.test_change_view -v`
Expected: PASS (6 tests).

- [ ] **Step 6: Commit**

```bash
git add snap/hooks/change-view-weather-admin tests/hooks/test_change_view.py
git commit -m "feat(hooks): change-view validates confdb writes before commit"
```

---

## Task 11: observe-view and connect hooks

**Files:**
- Create: `snap/hooks/observe-view-weather-admin`
- Create: `snap/hooks/connect-plug-weather-admin`

> `observe-view` fires when an **external** snap commits a write to the view (it does not fire for the snap that made the write — the configure hook handles that case in Task 9). `connect-plug` fires on connection and must **not** access confdb. Both only restart the daemon and always exit 0.

- [ ] **Step 1: Write the observe-view hook**

`snap/hooks/observe-view-weather-admin`:
```sh
#!/bin/sh -e

logger -t "weatherstation.observe-view" "confdb weather view changed; restarting daemon"

if snapctl services weatherstation.weatherstationd | grep -q active; then
    snapctl restart weatherstation.weatherstationd || true
fi

exit 0
```

- [ ] **Step 2: Write the connect-plug hook**

`snap/hooks/connect-plug-weather-admin`:
```sh
#!/bin/sh -e

logger -t "weatherstation.connect-plug" "weather-admin confdb interface connected"

# Do NOT access confdb here (would deadlock). Only restart so the daemon
# re-reads config on next start.
if snapctl services weatherstation.weatherstationd | grep -q active; then
    snapctl restart weatherstation.weatherstationd || true
fi

exit 0
```

- [ ] **Step 3: Make both hooks executable**

Run: `chmod +x snap/hooks/observe-view-weather-admin snap/hooks/connect-plug-weather-admin`
Expected: execute bit set on both.

- [ ] **Step 4: Lint the shell scripts**

Run: `sh -n snap/hooks/observe-view-weather-admin && sh -n snap/hooks/connect-plug-weather-admin && echo OK`
Expected: `OK` (no syntax errors).

- [ ] **Step 5: Commit**

```bash
git add snap/hooks/observe-view-weather-admin snap/hooks/connect-plug-weather-admin
git commit -m "feat(hooks): observe-view and connect-plug restart the daemon"
```

---

## Task 12: README configuration and deployment docs

**Files:**
- Modify: `README.md`

> Expand the README with the full setup sequence, including confdb connection order (schema ack → install custodian → connect plug → `snap set`) and the unix-socket content-interface requirement.

- [ ] **Step 1: Replace `README.md` with the full version**

```markdown
# weather-station-snap

A Core26 snap that polls the OpenWeather current-weather API on a configurable
interval and publishes a normalized JSON reading to an MQTT topic over either a
TCP or unix-domain-socket broker connection. All configuration is stored in
snapd confdb; the snap is its own confdb custodian.

## Published payload

    {"temp_c":18.4,"humidity":72,"pressure":1012,"wind_speed":3.2,"conditions":"Clouds","ts":"2026-06-24T10:00:00Z"}

## Setup

1. Import the confdb schema assertion (signed separately):

       sudo snap ack weather-confdb-schema.assert

2. Install the snap and connect its custodian plug:

       sudo snap install weatherstation
       sudo snap connect weatherstation:weather-admin

3. Configure it:

       sudo snap set weatherstation api-key=YOUR_KEY
       sudo snap set weatherstation lat=51.4545 lon=-2.5879
       sudo snap set weatherstation poll-interval=600
       sudo snap set weatherstation mqtt.server=tcp://broker.local mqtt.port=1883
       sudo snap set weatherstation mqtt.topic=weather/bristol

   The configure hook writes the complete configuration to confdb and restarts
   the daemon. Partial configuration is never written.

## Unix-socket broker

To publish over a unix domain socket instead of TCP, the broker snap must share
the directory containing its socket into this snap via the `mqtt-socket`
content interface:

    sudo snap connect weatherstation:mqtt-socket <broker-snap>:mqtt-socket

Then point the server at the socket path (port is ignored for unix):

    sudo snap set weatherstation mqtt.server=unix:///var/snap/weatherstation/current/mqtt/mqtt.sock
    sudo snap set weatherstation mqtt.topic=weather/bristol

## Configuration reference

| Key | Type | Notes |
|-----|------|-------|
| `api-key` | string | OpenWeather API key (stored as a confdb secret) |
| `lat` | number | Latitude, -90..90 |
| `lon` | number | Longitude, -180..180 |
| `poll-interval` | int | Seconds between polls, minimum 60 |
| `mqtt.server` | string | `tcp://host` or `unix:///abs/path.sock` |
| `mqtt.port` | int | 1..65535, used for tcp only |
| `mqtt.topic` | string | Topic to publish to |
```

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: full setup, unix-socket, and config reference"
```

---

## Task 13: Build, sign/import schema, and local integration test

**Files:** none created; this validates the end-to-end build and runtime.

> This task requires a Linux host with `snapcraft`, `snapd`, and `mosquitto` available. The confdb feature must be enabled and a signing key registered. Replace every `<ACCOUNT_ID>` placeholder with the real account ID before this task.

- [ ] **Step 1: Run the full Go test suite and build**

Run: `go test ./... && go build ./...`
Expected: all tests PASS; build succeeds.

- [ ] **Step 2: Build the snap**

Run: `snapcraft -v`
Expected: a `weatherstation_0.1.0_*.snap` file is produced in the project root.

- [ ] **Step 3: Enable confdb and register a signing key (one-time, if not already done)**

Run:
```bash
sudo snap set system experimental.confdb=true
snap known --remote account-key public-key-sha3-384=$(snapcraft list-keys 2>/dev/null | head -1) >/dev/null 2>&1 || true
```
Expected: confdb experimental feature enabled. (Key registration is environment-specific; follow the confdb getting-started guidance for your account.)

- [ ] **Step 4: Assemble, sign, and ack the confdb-schema assertion**

Build the assertion headers around `schema/weather-confdb-schema.yaml` (account-id `<ACCOUNT_ID>`, name `weather`, the storage/views body), sign it with your registered key, then:

Run: `sudo snap ack weather-confdb-schema.assert`
Expected: no error; `snap known confdb-schema` lists the `weather` schema.

- [ ] **Step 5: Install the snap (dangerous mode for local build) and connect plugs**

Run:
```bash
sudo snap install --dangerous ./weatherstation_0.1.0_*.snap
sudo snap connect weatherstation:weather-admin
```
Expected: snap installed; `snap connections weatherstation` shows `weather-admin` connected.

- [ ] **Step 6: Start a local mosquitto broker with both TCP and a unix socket**

Create `/tmp/mosquitto-test.conf`:
```
listener 1883
allow_anonymous true
listener 0 /tmp/mqtt-test.sock
```
Run: `mosquitto -c /tmp/mosquitto-test.conf &`
Expected: broker listening on TCP 1883 and unix socket `/tmp/mqtt-test.sock`.

- [ ] **Step 7: Configure for TCP and verify a message is published**

Run in one terminal:
```bash
mosquitto_sub -h localhost -p 1883 -t 'weather/bristol' -v
```
Run in another:
```bash
sudo snap set weatherstation api-key=YOUR_KEY lat=51.4545 lon=-2.5879 \
  poll-interval=60 mqtt.server=tcp://localhost mqtt.port=1883 mqtt.topic=weather/bristol
```
Expected: within a few seconds the subscriber prints a line like
`weather/bristol {"temp_c":...,"humidity":...,...,"ts":"..."}`.

- [ ] **Step 8: Verify daemon logs and confdb contents**

Run:
```bash
sudo snap logs weatherstation -n 20
snapctl get :weather-admin --view weather 2>/dev/null || sudo snap get weatherstation
```
Expected: logs show `published reading to weather/bristol`; config values are present.

- [ ] **Step 9: Confirm validation rejects a bad write**

Run: `sudo snap set weatherstation poll-interval=5`
Expected: the command fails (change-view hook aborts the transaction) with a message referencing `poll-interval`; confdb retains the previous valid value.

- [ ] **Step 10: Clean up the test broker**

Run: `pkill mosquitto; rm -f /tmp/mqtt-test.sock /tmp/mosquitto-test.conf`
Expected: broker stopped, socket removed.

- [ ] **Step 11: Final commit / tag**

```bash
git add -A
git commit -m "test: end-to-end build and integration validated" --allow-empty
git tag v0.1.0
```

---

## Future Enhancements (out of scope)

- **Hot reload (Approach 2):** have `observe-view`/configure signal the running daemon (SIGHUP) to re-read confdb and reconnect in place, avoiding the publish gap on a restart.
- **Broker auth/TLS:** add `username`, `password` (`visibility: secret`), and a TLS toggle/CA-path under the same `v1` schema; extend `MQTTConfig` and `publisher.New`.
- **Multiple observer snaps:** the `weather-state` read-only view already supports downstream consumers reacting to config via their own `observe-view` hooks.
