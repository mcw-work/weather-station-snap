import importlib.util
import os
import unittest
from importlib.machinery import SourceFileLoader
from unittest import mock

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

    def test_redacted_api_key_ok(self):
        # api-key is a confdb secret; snapd redacts it from the values the
        # change-view hook reads back, so an absent api-key must not be rejected.
        mod = load_hook()
        d = valid_doc()
        del d["api-key"]
        self.assertIsNone(mod.validate(d))

    def test_partial_doc_ok(self):
        # The admin may set values one key at a time; a partial document with
        # only some fields present must be accepted so setup can proceed.
        mod = load_hook()
        self.assertIsNone(mod.validate({"poll-interval": 600}))

    def test_empty_doc_ok(self):
        mod = load_hook()
        self.assertIsNone(mod.validate({}))

    def test_absent_server_ok(self):
        mod = load_hook()
        d = valid_doc()
        del d["mqtt"]["server"]
        self.assertIsNone(mod.validate(d))

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


class TestReadIncoming(unittest.TestCase):
    def _run(self, stdout):
        mod = load_hook()
        completed = mock.Mock(stdout=stdout)
        with mock.patch.object(mod.subprocess, "run", return_value=completed):
            return mod.read_incoming()

    def test_unwraps_weather_envelope(self):
        # snapctl keys the result by the requested view path.
        doc = self._run('{"weather": {"poll-interval": 600}}')
        self.assertEqual(doc, {"poll-interval": 600})

    def test_passthrough_when_not_wrapped(self):
        doc = self._run('{"poll-interval": 600}')
        self.assertEqual(doc, {"poll-interval": 600})

    def test_empty_output_is_empty_dict(self):
        self.assertEqual(self._run(""), {})


if __name__ == "__main__":
    unittest.main()
