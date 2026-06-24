#!/usr/bin/env python3
"""Generate the `snap sign` input JSON for the weather confdb-schema assertion.

Reads weather-confdb-schema.yaml (the source of truth for storage + views) and
emits a JSON document whose `body` header is the schema encoded as a JSON string,
ready to pipe into:

    snap sign -k <key-name> < weather-confdb-schema.sign.json > weather-confdb-schema.assert
    sudo snap ack weather-confdb-schema.assert
"""
import datetime
import json
import os
import sys

import yaml

ACCOUNT_ID = "bpyPt7Qr2Qbui3MJMgyzZ3WaQkyj6OkU"
NAME = "weather"

HERE = os.path.dirname(os.path.abspath(__file__))
SCHEMA_YAML = os.path.join(HERE, "weather-confdb-schema.yaml")
OUT_JSON = os.path.join(HERE, "weather-confdb-schema.sign.json")


def main():
    with open(SCHEMA_YAML) as f:
        schema = yaml.safe_load(f)

    body = json.dumps(schema, indent=2, sort_keys=False)

    assertion = {
        "type": "confdb-schema",
        "authority-id": ACCOUNT_ID,
        "account-id": ACCOUNT_ID,
        "name": NAME,
        "timestamp": datetime.datetime.now(datetime.timezone.utc)
        .replace(microsecond=0)
        .isoformat()
        .replace("+00:00", "Z"),
        "body": body,
    }

    out = json.dumps(assertion, indent=2)
    with open(OUT_JSON, "w") as f:
        f.write(out + "\n")
    sys.stdout.write(out + "\n")


if __name__ == "__main__":
    main()
