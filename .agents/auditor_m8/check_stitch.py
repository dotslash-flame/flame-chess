import os

p = "/Users/hriday/Downloads/stitch 2"
out_file = "/Users/hriday/Documents/Flame Chess/.agents/auditor_m8/stitch_output.txt"

with open(out_file, "w") as out:
    out.write(f"Exists: {os.path.exists(p)}\n")
    if os.path.exists(p):
        out.write(f"Listdir: {os.listdir(p)}\n")
        for entry in sorted(os.listdir(p)):
            sub = os.path.join(p, entry)
            if os.path.isdir(sub):
                out.write(f"Subdir {entry}: {os.listdir(sub)}\n")
