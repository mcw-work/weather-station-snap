# weather-station-snap

A Core26 snap that polls the OpenWeather current-weather API on a configurable
interval and publishes a normalized JSON reading to an MQTT topic over either a
TCP or unix-domain-socket broker connection. All configuration is stored in
snapd confdb; the snap is its own confdb custodian.

## Published payload

    {"temp_c":18.4,"humidity":72,"pressure":1012,"wind_speed":3.2,"conditions":"Clouds","ts":"2026-06-24T10:00:00Z"}

## Setup

Configuration is written directly to the snapd confdb `admin` view, not to
plain snap config. The view is addressed as `<account-id>/weather/admin`; the
commands below use the account that signed the schema assertion.

1. Import the confdb schema assertion (signed separately):

       sudo snap ack weather-confdb-schema.assert

2. Install the snap and connect its custodian plug:

       sudo snap install weatherstation
       sudo snap connect weatherstation:weather-admin

3. Configure it by writing to the confdb `admin` view. Each key can be set
   individually; the daemon stays inactive until the configuration is complete
   and starts automatically once every required value is present:

       ACCOUNT=bpyPt7Qr2Qbui3MJMgyzZ3WaQkyj6OkU
       sudo snap set $ACCOUNT/weather/admin weather.api-key=YOUR_KEY
       sudo snap set $ACCOUNT/weather/admin weather.location.lat=51.4545 weather.location.lon=-2.5879
       sudo snap set $ACCOUNT/weather/admin weather.poll-interval=600
       sudo snap set $ACCOUNT/weather/admin weather.mqtt.server=tcp://broker.local weather.mqtt.port=1883
       sudo snap set $ACCOUNT/weather/admin weather.mqtt.topic=weather/bristol

   The `change-view` hook validates each write (rejecting out-of-range or
   malformed values), and the daemon refuses to start until the configuration
   is complete, so partial state is never acted upon.

## Unix-socket broker

To publish over a unix domain socket instead of TCP, connect the `mqtt-socket`
content interface to the snap that *provides* the broker socket slot (for
example `connect-agent`). That snap bind-mounts its socket directory into this
snap at `$SNAP_DATA/mqtt`:

    sudo snap connect weatherstation:mqtt-socket connect-agent:mqtt-socket

Then point the server at the shared socket path inside this snap (port is
ignored for unix). Use this snap's own `$SNAP_DATA/mqtt` path, not the broker's
private directory — strict confinement only exposes the bind-mounted copy:

    sudo snap set $ACCOUNT/weather/admin weather.mqtt.server=unix:///var/snap/weatherstation/current/mqtt/mqtt.sock
    sudo snap set $ACCOUNT/weather/admin weather.mqtt.topic=weather/home

## Building

### Cross-compiling for ARM64

To build a snap targeting ARM64 from an x86_64 host, use snapcraft's
`--platform` flag in destructive mode:

    snapcraft pack --destructive-mode --platform arm64

This cross-compiles the Go daemon for `linux/arm64` and produces an
`.snap` file suitable for installation on ARM64 devices (e.g. Raspberry Pi).
Requires snapcraft 8+ and a host that supports the build environment for the
target platform.

## Configuration reference

Keys are confdb request paths under the `admin` view, set with
`sudo snap set <account-id>/weather/admin <key>=<value>`.

| Key | Type | Notes |
|-----|------|-------|
| `weather.api-key` | string | OpenWeather API key (stored as a confdb secret) |
| `weather.location.lat` | number | Latitude, -90..90 |
| `weather.location.lon` | number | Longitude, -180..180 |
| `weather.poll-interval` | int | Seconds between polls, minimum 60 |
| `weather.mqtt.server` | string | `tcp://host` or `unix:///abs/path.sock` |
| `weather.mqtt.port` | int | 1..65535, used for tcp only |
| `weather.mqtt.topic` | string | Topic to publish to |
