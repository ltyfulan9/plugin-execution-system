#!/usr/bin/env python3
import json, sys
payload = json.load(sys.stdin)
inp = payload.get("input", {})
data = inp.get("data", {})
field = inp.get("field", "")
if not isinstance(data, dict) or not field:
    print(json.dumps({"success": False, "data": {}, "error": "input.data object and input.field are required", "metrics": {}}))
else:
    print(json.dumps({"success": True, "data": {"field": field, "value": data.get(field)}, "error": "", "metrics": {}}))
