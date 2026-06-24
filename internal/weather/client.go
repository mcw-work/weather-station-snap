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
