"""Stop hook: keep Claude working until vet, the architecture check and all tests pass.

A new acceptance test may fail while it waits for approval (feature skill, step 4):
when the only failures are TestTerminal subtests that are not in HEAD yet, the turn may end.
"""

import json
import re
import subprocess
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parents[2]
ACCEPTANCE_FILE = "terminal_test.go"
ACCEPTANCE_TEST = "TestTerminal"
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


def committed_acceptance_tests() -> str:
    result = subprocess.run(
        ["git", "show", f"HEAD:{ACCEPTANCE_FILE}"], cwd=REPO, capture_output=True, text=True
    )
    return result.stdout


def pending_acceptance_tests(output: str) -> list[str]:
    """Return the new acceptance subtests when they are the only failures, else an empty list."""
    failures = re.findall(r"^\s*--- FAIL: (\S+)", output, re.MULTILINE)
    committed = committed_acceptance_tests()
    pending = []
    for name in failures:
        if name == ACCEPTANCE_TEST:
            continue
        parent, _, subtest = name.partition("/")
        # go test prints spaces in subtest names as underscores.
        if parent != ACCEPTANCE_TEST or not subtest or f'"{subtest.replace("_", " ")}"' in committed:
            return []
        pending.append(name)
    return pending


def main() -> int:
    hook_input = json.load(sys.stdin)
    if hook_input.get("stop_hook_active"):
        return 0
    for name, command in CHECKS:
        result = subprocess.run(command, cwd=REPO, capture_output=True, text=True)
        if result.returncode != 0:
            output = result.stdout + result.stderr
            pending = pending_acceptance_tests(output) if name == "tests" else []
            if pending:
                message = "dither-cli: new acceptance tests still fail, awaiting approval: " + ", ".join(pending)
                print(json.dumps({"systemMessage": message}))
                return 0
            summary = first_failure(output)
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
