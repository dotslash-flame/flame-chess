import subprocess
import os

cwd = "/Users/hriday/Documents/Flame Chess/e2e"
out_file = "/Users/hriday/Documents/Flame Chess/.agents/auditor_m8/test_run.txt"

env = os.environ.copy()
env["HOME"] = "/tmp"

with open(out_file, "w") as f:
    f.write("=== PLAYWRIGHT TEST RUN ===\n")
    p = subprocess.run(["npx", "playwright", "test"], cwd=cwd, capture_output=True, text=True, env=env)
    f.write(p.stdout)
    f.write(p.stderr)
    f.write(f"\nExit Code: {p.returncode}\n")

print("Playwright test run completed.")
