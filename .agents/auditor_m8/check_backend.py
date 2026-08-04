import subprocess
import os

cwd = "/Users/hriday/Documents/Flame Chess"
out_file = "/Users/hriday/Documents/Flame Chess/.agents/auditor_m8/backend_check.txt"

env = os.environ.copy()
env["HOME"] = "/tmp"
env["GIT_CONFIG_NOSYSTEM"] = "1"
env["GIT_CONFIG_GLOBAL"] = "/dev/null"

with open(out_file, "w") as f:
    f.write("=== GIT STATUS -- CMD INTERNAL ===\n")
    p1 = subprocess.run(["git", "status", "--porcelain", "--", "cmd", "internal"], cwd=cwd, capture_output=True, text=True, env=env)
    f.write(p1.stdout)
    f.write(p1.stderr)
    f.write("\n=== GIT DIFF -- CMD INTERNAL ===\n")
    p2 = subprocess.run(["git", "diff", "HEAD", "--", "cmd", "internal"], cwd=cwd, capture_output=True, text=True, env=env)
    f.write(p2.stdout)
    f.write(p2.stderr)
    f.write("\n=== GIT LOG CMD INTERNAL ===\n")
    p3 = subprocess.run(["git", "log", "-n", "5", "--", "cmd", "internal"], cwd=cwd, capture_output=True, text=True, env=env)
    f.write(p3.stdout)
    f.write(p3.stderr)
    f.write("\n=== OVERALL GIT STATUS ===\n")
    p4 = subprocess.run(["git", "status", "--porcelain"], cwd=cwd, capture_output=True, text=True, env=env)
    f.write(p4.stdout)
    f.write(p4.stderr)

print("Check completed.")
