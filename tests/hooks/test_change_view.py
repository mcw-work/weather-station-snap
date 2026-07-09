import os
import stat
import subprocess
import tempfile
import unittest

HOOK = os.path.join(
    os.path.dirname(__file__), "..", "..", "snap", "hooks", "change-view-weather-admin"
)

# Mock snapctl. The hook invokes: snapctl get :weather-admin <key>
# so $3 is the requested key path.
_MOCK_SNAPCTL = """#!/bin/sh
case "$3" in
    weather.mqtt.server) printf '%s' "$MOCK_SERVER" ;;
    weather.mqtt.port)   printf '%s' "$MOCK_PORT" ;;
esac
"""


def run_hook(server="", port=""):
    """Run the bash hook with a mocked snapctl; return its exit code."""
    with tempfile.TemporaryDirectory() as tmpdir:
        mock_path = os.path.join(tmpdir, "snapctl")
        with open(mock_path, "w") as f:
            f.write(_MOCK_SNAPCTL)
        os.chmod(
            mock_path,
            stat.S_IRWXU | stat.S_IRGRP | stat.S_IXGRP | stat.S_IROTH | stat.S_IXOTH,
        )

        env = os.environ.copy()
        env["PATH"] = tmpdir + os.pathsep + env.get("PATH", "")
        env["MOCK_SERVER"] = str(server)
        env["MOCK_PORT"] = str(port)

        return subprocess.run(["bash", HOOK], env=env, capture_output=True).returncode


class TestChangeViewHook(unittest.TestCase):
    def test_valid_tcp(self):
        self.assertEqual(run_hook(server="tcp://broker.local", port="1883"), 0)

    def test_unix_server_ok(self):
        self.assertEqual(run_hook(server="unix:///run/mosquitto/mqtt.sock"), 0)

    def test_absent_server_ok(self):
        # Partial write with no server set yet must be accepted.
        self.assertEqual(run_hook(server=""), 0)

    def test_absent_port_with_tcp_ok(self):
        # Port not yet set during a partial tcp write is still valid.
        self.assertEqual(run_hook(server="tcp://broker.local", port=""), 0)

    def test_bad_scheme(self):
        self.assertEqual(run_hook(server="broker.local", port="1883"), 1)

    def test_non_integer_port(self):
        self.assertEqual(run_hook(server="tcp://broker.local", port="abc"), 1)

    def test_tcp_port_zero(self):
        self.assertEqual(run_hook(server="tcp://broker.local", port="0"), 1)

    def test_tcp_port_too_high(self):
        self.assertEqual(run_hook(server="tcp://broker.local", port="65536"), 1)

    def test_tcp_port_boundary_low(self):
        self.assertEqual(run_hook(server="tcp://broker.local", port="1"), 0)

    def test_tcp_port_boundary_high(self):
        self.assertEqual(run_hook(server="tcp://broker.local", port="65535"), 0)


if __name__ == "__main__":
    unittest.main()
