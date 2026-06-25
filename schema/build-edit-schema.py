#!/usr/bin/env python3
"""Generate the document for `snapcraft edit-confdb-schema <account-id> weather`.

Reads weather-confdb-schema.yaml (the source of truth for storage + views) and
emits the YAML document snapcraft's editor expects:

  - `views` as top-level structured headers
  - `body` as a YAML literal block holding the storage schema as JSON

snapcraft fills in type/authority-id/timestamp and signs on save, so those are
omitted here. Paste the generated weather-confdb-schema.edit.yaml into the editor
opened by:

    snapcraft edit-confdb-schema bpyPt7Qr2Qbui3MJMgyzZ3WaQkyj6OkU weather
"""
import json
import os

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
OUT_YAML = os.path.join(HERE, "weather-confdb-schema.edit.yaml")


def main():
    with open(SCHEMA_YAML) as f:
        schema = yaml.safe_load(f)

    # views become structured top-level headers; inject optional summaries.
    views = {}
    for view_name, view in schema["views"].items():
        entry = {}
        if view_name in VIEW_SUMMARIES:
            entry["summary"] = VIEW_SUMMARIES[view_name]
        entry["rules"] = view["rules"]
        views[view_name] = entry

    views_block = yaml.dump(
        {"views": views}, default_flow_style=False, sort_keys=False
    ).rstrip()

    # body holds only the storage schema, as a JSON literal block.
    body = json.dumps({"storage": schema["storage"]}, indent=2)
    body_block = "\n".join("  " + line for line in body.splitlines())

    out = (
        f"account-id: {ACCOUNT_ID}\n"
        f"name: {NAME}\n"
        f"summary: {SUMMARY}\n"
        "# The revision for this confdb-schema\n"
        "# revision: 1\n"
        f"{views_block}\n"
        "\n"
        "body: |-\n"
        f"{body_block}\n"
    )

    with open(OUT_YAML, "w") as f:
        f.write(out)
    print(out, end="")


if __name__ == "__main__":
    main()
