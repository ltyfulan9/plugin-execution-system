from __future__ import annotations

import json
import urllib.error
import urllib.request
from dataclasses import dataclass
from typing import Any, Dict, Iterable, Optional


class PSError(RuntimeError):
    def __init__(self, status: int, code: str, message: str):
        super().__init__(f"PES API error {status} {code}: {message}")
        self.status = status
        self.code = code
        self.message = message


@dataclass
class Client:
    base_url: str = "http://127.0.0.1:8080"
    token: str = "demo-token"
    timeout: float = 10.0

    def health(self) -> Dict[str, Any]:
        return self._request("GET", "/api/v1/health")

    def plugins(self, page: int = 1, page_size: int = 50) -> Dict[str, Any]:
        return self._request("GET", f"/api/v1/plugins?page={page}&page_size={page_size}")

    def create_execution(self, plugin_ids: Iterable[str], input_data: Dict[str, Any], idempotency_key: Optional[str] = None) -> Dict[str, Any]:
        headers = {}
        if idempotency_key:
            headers["Idempotency-Key"] = idempotency_key
        return self._request("POST", "/api/v1/executions", {"plugin_ids": list(plugin_ids), "input": input_data}, headers=headers)

    def execution(self, execution_id: str) -> Dict[str, Any]:
        return self._request("GET", f"/api/v1/executions/{execution_id}")

    def results(self, execution_id: str) -> Dict[str, Any]:
        return self._request("GET", f"/api/v1/executions/{execution_id}/results")

    def summary(self, execution_id: str) -> Dict[str, Any]:
        return self._request("GET", f"/api/v1/executions/{execution_id}/summary")

    def webhooks(self) -> Dict[str, Any]:
        return self._request("GET", "/api/v1/webhooks")

    def create_webhook(self, name: str, url: str, events: Optional[Iterable[str]] = None, secret: Optional[str] = None) -> Dict[str, Any]:
        body: Dict[str, Any] = {"name": name, "url": url}
        if events is not None:
            body["events"] = list(events)
        if secret:
            body["secret"] = secret
        return self._request("POST", "/api/v1/webhooks", body)

    def _request(self, method: str, path: str, body: Optional[Dict[str, Any]] = None, headers: Optional[Dict[str, str]] = None) -> Dict[str, Any]:
        data = None
        req_headers = {"Authorization": f"Bearer {self.token}"}
        if body is not None:
            data = json.dumps(body).encode("utf-8")
            req_headers["Content-Type"] = "application/json"
        if headers:
            req_headers.update(headers)
        req = urllib.request.Request(self.base_url.rstrip("/") + path, data=data, headers=req_headers, method=method)
        try:
            with urllib.request.urlopen(req, timeout=self.timeout) as resp:
                return json.loads(resp.read().decode("utf-8"))
        except urllib.error.HTTPError as exc:
            try:
                payload = json.loads(exc.read().decode("utf-8"))
                raise PSError(exc.code, payload.get("code", "UNKNOWN"), payload.get("message", str(exc))) from exc
            except json.JSONDecodeError as json_exc:
                raise PSError(exc.code, "HTTP_ERROR", str(exc)) from json_exc
