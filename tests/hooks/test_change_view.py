import importlib.util
import os
import unittest
from importlib.machinery import SourceFileLoader

HOOK = os.path.join(
    os.path.dirname(__file__), "..", "..", "snap", "hooks", "change-view-weather-admin"
)


def load_hook():
    # Snap hooks are extensionless, so an explicit loader is required.
    loader = SourceFileLoader("change_view_hook", HOOK)
    spec = importlib.util.spec_from_file_location("change_view_hook", HOOK, loader=loader)
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
