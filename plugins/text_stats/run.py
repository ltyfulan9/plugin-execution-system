#!/usr/bin/env python3
import json, sys
payload = json.load(sys.stdin)
text = str(payload.get("input", {}).get("text", ""))
print(json.dumps({"success": True, "data": {"chars": len(text), "words": len(text.split()), "lines": 0 if text == "" else text.count("\n") + 1}, "error": "", "metrics": {}}))
