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
