#!/usr/bin/env python3
"""Run the full Plugin Execution System demo flow against a running server.

Usage:
  BASE_URL=http://localhost:8080 python3 scripts/demo_flow.py

The script uses only Python standard library.
"""

import json
import os
import sys
import time
import urllib.error
import urllib.request

BASE_URL = os.environ.get("BASE_URL", "http://localhost:8080").rstrip("/")
ADMIN_TOKEN = os.environ.get("ADMIN_TOKEN", "admin-token")
DEMO_TOKEN = os.environ.get("DEMO_TOKEN", "demo-token")
IDEMPOTENCY_KEY = os.environ.get("IDEMPOTENCY_KEY", f"demo-flow-{int(time.time())}")
INCLUDE_ERROR = os.environ.get("INCLUDE_ERROR", "0") == "1"


def request(method, path, token=None, body=None, headers=None):
    raw = None
    h = {"Accept": "application/json"}
    if body is not None:
        raw = json.dumps(body).encode("utf-8")
        h["Content-Type"] = "application/json"
    if token:
        h["Authorization"] = f"Bearer {token}"
    if headers:
        h.update(headers)
    req = urllib.request.Request(BASE_URL + path, data=raw, method=method, headers=h)
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            payload = json.loads(resp.read().decode("utf-8"))
            return resp.status, payload
    except urllib.error.HTTPError as exc:
        payload = json.loads(exc.read().decode("utf-8"))
        return exc.code, payload


def require_ok(status, payload, label):
    if status < 200 or status >= 300 or payload.get("code") != "OK":
        raise SystemExit(f"{label} failed: status={status} payload={json.dumps(payload, ensure_ascii=False, indent=2)}")
    return payload.get("data")


def main():
    print(f"base_url={BASE_URL}")

    status, payload = request("GET", "/api/health")
    require_ok(status, payload, "health")

    status, payload = request("POST", "/api/plugins/reload", token=ADMIN_TOKEN)
    data = require_ok(status, payload, "reload plugins")
    print("reload:", data)

    status, payload = request("GET", "/api/plugins", token=ADMIN_TOKEN)
    plugins = require_ok(status, payload, "list plugins")
    by_name = {p["name"]: p for p in plugins}
    wanted = ["echo", "text_stats"]
    if INCLUDE_ERROR:
        wanted.append("error_demo")
    missing = [name for name in wanted if name not in by_name]
    if missing:
        raise SystemExit(f"missing plugins: {missing}")

    plugin_ids = []
    for name in wanted:
        plugin = by_name[name]
        status, payload = request("POST", f"/api/plugins/{plugin['id']}/enable", token=ADMIN_TOKEN)
        enabled = require_ok(status, payload, f"enable {name}")
        plugin_ids.append(enabled["id"])
        print("enabled:", enabled["name"], enabled["status"])

    body = {
        "plugin_ids": plugin_ids,
        "input": {
            "text": "hello plugin execution system",
            "data": {"name": "satoyuki", "lang": "go"},
            "field": "name",
        },
    }
    status, payload = request(
        "POST",
        "/api/executions",
        token=DEMO_TOKEN,
        body=body,
        headers={"Idempotency-Key": IDEMPOTENCY_KEY},
    )
    execution = require_ok(status, payload, "create execution")
    execution_id = execution["id"]
    print("execution:", execution_id, execution["status"])

    final_statuses = {"Success", "PartialSuccess", "Failed", "Timeout", "Canceled"}
    for _ in range(100):
        status, payload = request("GET", f"/api/executions/{execution_id}", token=DEMO_TOKEN)
        execution = require_ok(status, payload, "get execution")
        print("status:", execution["status"])
        if execution["status"] in final_statuses:
            break
        time.sleep(0.1)
    else:
        raise SystemExit("execution did not finish in time")

    status, payload = request("GET", f"/api/executions/{execution_id}/summary", token=DEMO_TOKEN)
    summary = require_ok(status, payload, "summary")
    print("summary:", json.dumps(summary, ensure_ascii=False, indent=2))

    status, payload = request("GET", f"/api/executions/{execution_id}/results", token=DEMO_TOKEN)
    results = require_ok(status, payload, "results")
    print("results:", json.dumps(results, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"demo failed: {exc}", file=sys.stderr)
        raise
