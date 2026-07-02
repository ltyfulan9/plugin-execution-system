#!/usr/bin/env python3
import json, sys
_ = json.load(sys.stdin)
print(json.dumps({"success": False, "data": {}, "error": "intentional error from error_demo", "metrics": {}}))
