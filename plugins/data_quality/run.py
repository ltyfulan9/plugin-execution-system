#!/usr/bin/env python3
import hashlib
import json
import sys


def normalize_record(record):
    return json.dumps(record, sort_keys=True, separators=(",", ":"), ensure_ascii=False)


def main():
    payload = json.load(sys.stdin)
    input_data = payload.get("input", {})
    records = input_data.get("records", [])
    required_fields = input_data.get("required_fields", [])

    if not isinstance(records, list):
        print(json.dumps({"success": False, "data": {}, "error": "input.records must be an array", "metrics": {}}))
        return
    if not isinstance(required_fields, list):
        print(json.dumps({"success": False, "data": {}, "error": "input.required_fields must be an array", "metrics": {}}))
        return

    missing_by_field = {str(field): 0 for field in required_fields}
    empty_string_fields = {}
    invalid_records = 0
    fingerprints = {}

    for record in records:
        if not isinstance(record, dict):
            invalid_records += 1
            continue
        for field in required_fields:
            key = str(field)
            value = record.get(key)
            if value is None:
                missing_by_field[key] += 1
            elif isinstance(value, str) and value.strip() == "":
                missing_by_field[key] += 1
                empty_string_fields[key] = empty_string_fields.get(key, 0) + 1
        digest = hashlib.sha256(normalize_record(record).encode("utf-8")).hexdigest()
        fingerprints[digest] = fingerprints.get(digest, 0) + 1

    duplicate_rows = sum(count - 1 for count in fingerprints.values() if count > 1)
    total_missing = sum(missing_by_field.values())
    quality_score = 100
    if records:
        penalty_units = total_missing + invalid_records + duplicate_rows
        quality_score = max(0, round(100 - (penalty_units / max(len(records), 1) * 20), 2))

    print(json.dumps({
        "success": True,
        "data": {
            "record_count": len(records),
            "invalid_records": invalid_records,
            "required_fields": [str(field) for field in required_fields],
            "missing_by_field": missing_by_field,
            "empty_string_fields": empty_string_fields,
            "duplicate_rows": duplicate_rows,
            "quality_score": quality_score
        },
        "error": "",
        "metrics": {"records": len(records), "missing_values": total_missing, "duplicates": duplicate_rows}
    }, ensure_ascii=False))


if __name__ == "__main__":
    main()
