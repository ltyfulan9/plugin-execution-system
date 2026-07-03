#!/usr/bin/env python3
"""Start PES in local/dev mode and verify the complete executable closed loop.

This is intentionally not a new feature. It is a runnable acceptance proof:
server -> plugin reload -> enable -> create execution -> worker -> runtime ->
results -> summary -> events -> attempts -> audit -> metrics.

It uses only Python standard library and a temporary local metadata directory.
"""

from __future__ import annotations

import json
import os
import shutil
import signal
import socket
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
ADMIN_TOKEN = os.environ.get("ADMIN_TOKEN", "admin-token")
DEMO_TOKEN = os.environ.get("DEMO_TOKEN", "demo-token")
FINAL_STATUSES = {"Success", "PartialSuccess", "Failed", "Timeout", "Canceled"}


def free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.bind(("127.0.0.1", 0))
        return int(s.getsockname()[1])


def http_json(method: str, base_url: str, path: str, token: str | None = None, body=None, headers=None):
    raw = None
    request_headers = {"Accept": "application/json", "Trace-ID": "verify-closed-loop"}
    if body is not None:
        raw = json.dumps(body).encode("utf-8")
        request_headers["Content-Type"] = "application/json"
    if token:
        request_headers["Authorization"] = f"Bearer {token}"
    if headers:
        request_headers.update(headers)
    req = urllib.request.Request(base_url + path, data=raw, method=method, headers=request_headers)
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            payload = json.loads(resp.read().decode("utf-8"))
            return resp.status, payload
    except urllib.error.HTTPError as exc:
        try:
            payload = json.loads(exc.read().decode("utf-8"))
        except Exception:
            payload = {"code": "HTTP_ERROR", "message": str(exc)}
        return exc.code, payload


def http_text(base_url: str, path: str) -> str:
    with urllib.request.urlopen(base_url + path, timeout=15) as resp:
        return resp.read().decode("utf-8", errors="replace")


def expect_ok(status: int, payload: dict, label: str, statuses=(200, 201)):
    if status not in statuses or payload.get("code") != "OK":
        raise AssertionError(f"{label} failed: status={status} payload={json.dumps(payload, ensure_ascii=False, indent=2)}")
    return payload.get("data")


def expect_error(status: int, payload: dict, label: str, want_status: int, want_code: str):
    if status != want_status or payload.get("code") != want_code:
        raise AssertionError(f"{label} expected {want_status}/{want_code}, got status={status} payload={payload}")


def wait_livez(base_url: str, proc: subprocess.Popen, log_path: Path, timeout_seconds: float = 30.0) -> None:
    deadline = time.time() + timeout_seconds
    last_error = None
    while time.time() < deadline:
        if proc.poll() is not None:
            logs = log_path.read_text(errors="replace") if log_path.exists() else ""
            raise RuntimeError(f"server exited early with code={proc.returncode}\n--- server logs ---\n{logs}")
        try:
            status, payload = http_json("GET", base_url, "/livez")
            if status == 200 and payload.get("code") == "OK":
                return
        except Exception as exc:  # server not ready yet
            last_error = exc
        time.sleep(0.2)
    logs = log_path.read_text(errors="replace") if log_path.exists() else ""
    raise TimeoutError(f"server did not become live: {last_error}\n--- server logs ---\n{logs}")


def wait_execution(base_url: str, execution_id: str, timeout_seconds: float = 10.0) -> dict:
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        status, payload = http_json("GET", base_url, f"/api/v1/executions/{execution_id}", DEMO_TOKEN)
        data = expect_ok(status, payload, "get execution")
        if data["status"] in FINAL_STATUSES:
            return data
        time.sleep(0.1)
    raise TimeoutError(f"execution {execution_id} did not finish in {timeout_seconds}s")


def plugins_by_name(base_url: str) -> dict:
    status, payload = http_json("GET", base_url, "/api/v1/plugins", ADMIN_TOKEN)
    plugins = expect_ok(status, payload, "list plugins")
    return {p["name"]: p for p in plugins}


def enable(base_url: str, plugin: dict) -> dict:
    status, payload = http_json("POST", base_url, f"/api/v1/plugins/{plugin['id']}/enable", ADMIN_TOKEN)
    return expect_ok(status, payload, f"enable {plugin['name']}")


def create_execution(base_url: str, plugin_ids: list[str], input_obj: dict, idem_key: str) -> dict:
    body = {"plugin_ids": plugin_ids, "input": input_obj}
    status, payload = http_json(
        "POST",
        base_url,
        "/api/v1/executions",
        DEMO_TOKEN,
        body=body,
        headers={"Idempotency-Key": idem_key},
    )
    return expect_ok(status, payload, "create execution", statuses=(201,))


def sample_input() -> dict:
    return {
        "text": "hello plugin execution system with audit evidence",
        "data": {"name": "satoyuki"},
        "field": "name",
        "keywords": ["plugin", "execution", "audit", "missing-keyword"],
        "records": [
            {"id": 1, "name": "alpha", "email": "alpha@example.com"},
            {"id": 2, "name": "", "email": "beta@example.com"},
            {"id": 2, "name": "", "email": "beta@example.com"},
            {"id": 3, "email": None},
        ],
        "required_fields": ["id", "name", "email"],
    }


def verify_success_flow(base_url: str, plugins: list[dict]) -> str:
    plugin_ids = [plugin["id"] for plugin in plugins]
    created = create_execution(
        base_url,
        plugin_ids,
        sample_input(),
        "verify-success",
    )
    repeated = create_execution(
        base_url,
        plugin_ids,
        sample_input(),
        "verify-success",
    )
    if repeated["id"] != created["id"]:
        raise AssertionError(f"idempotency failed: first={created['id']} repeated={repeated['id']}")

    conflict_body = {"plugin_ids": plugin_ids, "input": {"text": "different"}}
    status, payload = http_json(
        "POST",
        base_url,
        "/api/v1/executions",
        DEMO_TOKEN,
        body=conflict_body,
        headers={"Idempotency-Key": "verify-success"},
    )
    expect_error(status, payload, "idempotency conflict", 409, "IDEMPOTENCY_CONFLICT")

    final = wait_execution(base_url, created["id"])
    if final["status"] != "Success":
        raise AssertionError(f"success flow expected Success, got {final}")

    status, payload = http_json("GET", base_url, f"/api/v1/executions/{created['id']}/summary", DEMO_TOKEN)
    summary = expect_ok(status, payload, "summary")
    if summary["status"] != "Success" or summary["total"] != len(plugin_ids) or summary["success"] != len(plugin_ids):
        raise AssertionError(f"bad success summary: {summary}")

    status, payload = http_json("GET", base_url, f"/api/v1/executions/{created['id']}/results", DEMO_TOKEN)
    results = expect_ok(status, payload, "results")
    if len(results) != len(plugin_ids) or any(r["status"] != "Success" for r in results):
        raise AssertionError(f"bad success results: {results}")
    result_names = {r["plugin_name"] for r in results}
    for expected in ["data_quality", "keyword_audit"]:
        if expected not in result_names:
            raise AssertionError(f"missing new plugin result {expected}: {result_names}")

    status, payload = http_json("GET", base_url, f"/api/v1/executions/{created['id']}/events", DEMO_TOKEN)
    events = expect_ok(status, payload, "events")
    event_types = {e["type"] for e in events}
    required = {"ExecutionCreated", "ExecutionQueued", "ExecutionStarted", "PluginStarted", "PluginFinished", "ExecutionFinished"}
    if not required.issubset(event_types):
        raise AssertionError(f"missing event types: required={required} got={event_types}")

    status, payload = http_json("GET", base_url, f"/api/v1/executions/{created['id']}/attempts", DEMO_TOKEN)
    attempts = expect_ok(status, payload, "attempts")
    if len(attempts) < 1 or attempts[0]["status"] != "Success":
        raise AssertionError(f"bad attempts: {attempts}")

    status, payload = http_json("GET", base_url, f"/api/v1/audit/executions/{created['id']}", ADMIN_TOKEN)
    audit = expect_ok(status, payload, "execution audit")
    if not audit:
        raise AssertionError("expected non-empty audit trail")

    return created["id"]


def verify_partial_flow(base_url: str, echo: dict, error_demo: dict) -> str:
    created = create_execution(base_url, [echo["id"], error_demo["id"]], {"text": "partial"}, "verify-partial")
    final = wait_execution(base_url, created["id"])
    if final["status"] != "PartialSuccess":
        raise AssertionError(f"partial flow expected PartialSuccess, got {final}")

    status, payload = http_json("GET", base_url, f"/api/v1/executions/{created['id']}/summary", DEMO_TOKEN)
    summary = expect_ok(status, payload, "partial summary")
    if summary["status"] != "PartialSuccess" or summary["success"] != 1 or summary["failed"] != 1:
        raise AssertionError(f"bad partial summary: {summary}")
    return created["id"]


def main() -> int:
    port = int(os.environ.get("VERIFY_PORT", free_port()))
    base_url = f"http://127.0.0.1:{port}"
    temp_root = Path(tempfile.mkdtemp(prefix="pes-verify-"))
    store_dir = temp_root / "data"
    log_path = temp_root / "server.log"
    env = os.environ.copy()
    env.update(
        {
            "APP_MODE": "dev",
            "METADATA_STORE": "local-json",
            "ALLOW_LOCAL_JSON_STORE": "true",
            "STORAGE_DIR": str(store_dir),
            "SERVER_ADDR": f"127.0.0.1:{port}",
            "PLUGIN_DIR": str(ROOT / "plugins"),
            "ADMIN_TOKEN": ADMIN_TOKEN,
            "DEMO_TOKEN": DEMO_TOKEN,
            "PLUGIN_CONTAINER_RUNTIME_ENABLED": "false",
        }
    )

    print("== PES closed-loop verification ==")
    print(f"repo={ROOT}")
    print(f"base_url={base_url}")
    print(f"storage_dir={store_dir}")

    with log_path.open("w") as log_file:
        proc = subprocess.Popen(
            ["go", "run", "./cmd/server"],
            cwd=str(ROOT),
            env=env,
            stdout=log_file,
            stderr=subprocess.STDOUT,
            text=True,
        )
        try:
            wait_livez(base_url, proc, log_path)
            print("[ok] server live")

            for path in ["/readyz", "/workerz", "/dependencyz"]:
                status, payload = http_json("GET", base_url, path)
                expect_ok(status, payload, path)
                print(f"[ok] {path}")

            status, payload = http_json("POST", base_url, "/api/v1/plugins/reload", ADMIN_TOKEN)
            expect_ok(status, payload, "reload plugins")
            print("[ok] plugin reload")

            by_name = plugins_by_name(base_url)
            for name in ["echo", "text_stats", "error_demo", "data_quality", "keyword_audit"]:
                if name not in by_name:
                    raise AssertionError(f"missing plugin {name}")
            echo = enable(base_url, by_name["echo"])
            text_stats = enable(base_url, by_name["text_stats"])
            error_demo = enable(base_url, by_name["error_demo"])
            data_quality = enable(base_url, by_name["data_quality"])
            keyword_audit = enable(base_url, by_name["keyword_audit"])
            print("[ok] plugins enabled: echo, text_stats, error_demo, data_quality, keyword_audit")

            success_id = verify_success_flow(base_url, [echo, text_stats, data_quality, keyword_audit])
            print(f"[ok] success execution closed loop: {success_id}")

            partial_id = verify_partial_flow(base_url, echo, error_demo)
            print(f"[ok] partial-success execution closed loop: {partial_id}")

            status, payload = http_json("GET", base_url, "/api/v1/executions?page=1&page_size=10", DEMO_TOKEN)
            page = expect_ok(status, payload, "execution pagination")
            if page.get("total", 0) < 2 or len(page.get("items", [])) < 2:
                raise AssertionError(f"bad execution page: {page}")
            print("[ok] pagination")

            metrics = http_text(base_url, "/metrics")
            for marker in ["pes_task_submitted_total", "pes_plugin_completed_total", "pes_idempotency_hits_total"]:
                if marker not in metrics:
                    raise AssertionError(f"missing metric {marker}")
            print("[ok] metrics")

            print("\nVERIFICATION PASSED")
            return 0
        finally:
            if proc.poll() is None:
                proc.send_signal(signal.SIGTERM)
                try:
                    proc.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    proc.kill()
                    proc.wait(timeout=5)
            if os.environ.get("KEEP_VERIFY_DATA") != "1":
                shutil.rmtree(temp_root, ignore_errors=True)
            else:
                print(f"kept verification data at {temp_root}")


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:
        print(f"VERIFICATION FAILED: {exc}", file=sys.stderr)
        raise
