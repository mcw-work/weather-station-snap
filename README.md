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
