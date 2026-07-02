#!/usr/bin/env python3
import json, sys
payload = json.load(sys.stdin)
print(json.dumps({"success": True, "data": {"echo": payload.get("input", {})}, "error": "", "metrics": {}}))
