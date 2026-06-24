import importlib.util
import os
import unittest
from importlib.machinery import SourceFileLoader

HOOK = os.path.join(os.path.dirname(__file__), "..", "..", "snap", "hooks", "configure")


def load_hook():
    # Snap hooks are extensionless, so an explicit loader is required;
    # spec_from_file_location cannot infer one from the suffix.
    loader = SourceFileLoader("configure_hook", HOOK)
    spec = importlib.util.spec_from_file_location("configure_hook", HOOK, loader=loader)
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
