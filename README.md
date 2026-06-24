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
