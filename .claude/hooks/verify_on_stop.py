"""Stop hook: keep Claude working until vet, the architecture check and all tests pass."""

import json
import subprocess
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parents[2]
CHECKS = [
    ("vet", ["go", "vet", "./..."]),
    ("architecture", ["go", "test", "-count=1", "-run", "TestArchitecture", "."]),
    ("tests", ["go", "test", "-count=1", "./..."]),
]


def first_failure(output: str) -> str:
    lines = output.splitlines()
    for index, line in enumerate(lines):
        if line.lstrip().startswith("--- FAIL"):
            return "\n".join(lines[index : index + 3]).strip()
    return "\n".join(lines[:20]).strip()


def main() -> int:
    hook_input = json.load(sys.stdin)
    if hook_input.get("stop_hook_active"):
        return 0
    for name, command in CHECKS:
        result = subprocess.run(command, cwd=REPO, capture_output=True, text=True)
        if result.returncode != 0:
            summary = first_failure(result.stdout + result.stderr)
            print(f"dither-cli: {name} failed\n{summary}", file=sys.stderr)
            reason = (
                f"Verification failed for dither-cli ({name}: `{' '.join(command)}`).\n"
                f"{summary}\n\nFix the failure, then finish the turn."
            )
            print(json.dumps({"decision": "block", "reason": reason}))
            return 0
    return 0


if __name__ == "__main__":
    sys.exit(main())
