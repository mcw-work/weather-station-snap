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
