#!/usr/bin/env python3
import json
import re
import sys


def count_keyword(text, keyword, case_sensitive):
    flags = 0 if case_sensitive else re.IGNORECASE
    pattern = re.escape(str(keyword))
    return len(re.findall(pattern, text, flags))


def main():
    payload = json.load(sys.stdin)
    input_data = payload.get("input", {})
    text = str(input_data.get("text", ""))
    keywords = input_data.get("keywords", [])
    case_sensitive = bool(input_data.get("case_sensitive", False))

    if not isinstance(keywords, list):
        print(json.dumps({"success": False, "data": {}, "error": "input.keywords must be an array", "metrics": {}}))
        return

    frequencies = {}
    for keyword in keywords:
        key = str(keyword)
        if key:
            frequencies[key] = count_keyword(text, key, case_sensitive)

    present = [keyword for keyword, count in frequencies.items() if count > 0]
    missing = [keyword for keyword, count in frequencies.items() if count == 0]
    coverage = 100
    if frequencies:
        coverage = round(len(present) / len(frequencies) * 100, 2)

    print(json.dumps({
        "success": True,
        "data": {
            "keyword_count": len(frequencies),
            "present": present,
            "missing": missing,
            "frequencies": frequencies,
            "coverage_percent": coverage,
            "case_sensitive": case_sensitive
        },
        "error": "",
        "metrics": {"keywords": len(frequencies), "missing": len(missing)}
    }, ensure_ascii=False))


if __name__ == "__main__":
    main()
