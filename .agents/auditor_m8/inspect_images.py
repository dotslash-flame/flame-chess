import os
from PIL import Image

screenshots_dir = "/Users/hriday/Documents/Flame Chess/e2e/screenshots"
mockups_dir = "/Users/hriday/Downloads/stitch 2"
out_file = "/Users/hriday/Documents/Flame Chess/.agents/auditor_m8/images_report.txt"

with open(out_file, "w") as out:
    out.write("--- Captured Screenshots ---\n")
    for fname in sorted(os.listdir(screenshots_dir)):
        if fname.endswith(".png"):
            fpath = os.path.join(screenshots_dir, fname)
            im = Image.open(fpath)
            out.write(f"{fname}: {im.size}, mode={im.mode}, filesize={os.path.getsize(fpath)} bytes\n")

    out.write("\n--- Stitch 2 Mockups ---\n")
    for root, dirs, files in os.walk(mockups_dir):
        for f in files:
            if f.endswith(".png"):
                fpath = os.path.join(root, f)
                rel = os.path.relpath(fpath, mockups_dir)
                im = Image.open(fpath)
                out.write(f"{rel}: {im.size}, mode={im.mode}, filesize={os.path.getsize(fpath)} bytes\n")

print("Inspection completed.")
