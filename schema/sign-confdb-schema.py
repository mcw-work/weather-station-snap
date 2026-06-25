#!/usr/bin/env python3
"""Emit the weather confdb-schema assertion as JSON for `snap sign`.

The schema source of truth is weather-confdb-schema.yaml (storage + views).
This script renders it into the assertion document snapd expects, leaving the
signature to `snap sign`.

Usage (bump the revision on every change, then re-ack):

    python3 schema/sign-confdb-schema.py <revision> \
        | snap sign -k <key-name> > schema/weather-confdb-schema.assert
    sudo snap ack schema/weather-confdb-schema.assert
"""
import datetime
import json
import os
import sys

import yaml

ACCOUNT_ID = "bpyPt7Qr2Qbui3MJMgyzZ3WaQkyj6OkU"
NAME = "weather"
SUMMARY = "OpenWeather data published to MQTT, configured via confdb."
VIEW_SUMMARIES = {
    "admin": "Read-write control view used by the custodian snap.",
    "state": "Read-only view for observer snaps.",
}

HERE = os.path.dirname(os.path.abspath(__file__))
SCHEMA_YAML = os.path.join(HERE, "weather-confdb-schema.yaml")


def main():
    if len(sys.argv) != 2:
        sys.exit(f"usage: {sys.argv[0]} <revision>")
    revision = sys.argv[1]

    with open(SCHEMA_YAML) as f:
        schema = yaml.safe_load(f)

    views = {}
    for name, view in schema["views"].items():
        entry = {}
        if name in VIEW_SUMMARIES:
            entry["summary"] = VIEW_SUMMARIES[name]
        entry["rules"] = view["rules"]
        views[name] = entry

    timestamp = (
        datetime.datetime.now(datetime.timezone.utc)
        .replace(microsecond=0)
        .isoformat()
        .replace("+00:00", "Z")
    )

    assertion = {
        "type": "confdb-schema",
        "authority-id": ACCOUNT_ID,
        "account-id": ACCOUNT_ID,
        "name": NAME,
        "summary": SUMMARY,
        "revision": str(revision),
        "timestamp": timestamp,
        "views": views,
        "body": json.dumps({"storage": schema["storage"]}, sort_keys=True),
    }
    print(json.dumps(assertion))


if __name__ == "__main__":
    main()
