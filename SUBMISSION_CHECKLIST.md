# Submission Checklist

## Required Materials

- [x] Public GitHub repository
- [x] Go implementation
- [x] Runnable local demo
- [x] JSONL process record
- [x] README with run instructions
- [x] Tests and verification script

## Repository

- Repository: <https://github.com/ltyfulan9/plugin-execution-system>
- Branch: `main`
- Main language: Go
- Demo command: `make demo`

## Local Verification

Run these commands before submission:

```bash
go test ./...
go test -race ./...
make demo
```

On Windows systems without `make`, run the equivalent command:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/demo.ps1
```

Expected `make demo` ending:

```text
VERIFICATION PASSED
```

## JSONL Checks

- [ ] File name follows the required pattern.
- [ ] Each line is exactly one JSON object.
- [ ] `round_id` starts at 1 and is unique.
- [ ] `prompt_content` contains the user instruction for that round.
- [ ] `modify_diff` records the code or artifact changes for that round.
- [ ] `commit_hash` is filled with a real commit hash or clearly maps local iteration to the import commit.
- [ ] `modify_time` uses `YYYY-MM-DD HH:MM:SS`.
- [ ] `agent_type` follows the assessment requirement.
- [ ] `dev_language` is `Go`.

## Files to Submit

- GitHub repository URL.
- Final JSONL process record.
- Optional: short note explaining that early development was iterative locally and later consolidated into GitHub.

## Do Not Submit

- Old release zip files generated before Windows compatibility fixes.
- Local `data/` runtime state.
- Local environment files.
- Temporary verification folders.
