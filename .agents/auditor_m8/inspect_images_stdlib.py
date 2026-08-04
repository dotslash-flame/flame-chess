import os
import struct

def get_png_dimensions(filepath):
    try:
        with open(filepath, 'rb') as f:
            data = f.read(24)
            if len(data) >= 24 and data[:8] == b'\x89PNG\r\n\x1a\n':
                width, height = struct.unpack('>II', data[16:24])
                return width, height
    except Exception as e:
        return str(e), str(e)
    return None, None

screenshots_dir = "/Users/hriday/Documents/Flame Chess/e2e/screenshots"
mockups_dir = "/Users/hriday/Downloads/stitch 2"
out_file = "/Users/hriday/Documents/Flame Chess/.agents/auditor_m8/images_report.txt"

lines = []
lines.append("=== CAPTURED SCREENSHOTS IN e2e/screenshots ===")
if os.path.exists(screenshots_dir):
    for fname in sorted(os.listdir(screenshots_dir)):
        if fname.endswith(".png"):
            fpath = os.path.join(screenshots_dir, fname)
            w, h = get_png_dimensions(fpath)
            size = os.path.getsize(fpath)
            lines.append(f"{fname:25s}: {w}x{h} px, {size} bytes")

lines.append("\n=== STITCH 2 MOCKUPS IN /Users/hriday/Downloads/stitch 2 ===")
if os.path.exists(mockups_dir):
    for root, dirs, files in sorted(os.walk(mockups_dir)):
        for f in sorted(files):
            if f.endswith(".png"):
                fpath = os.path.join(root, f)
                rel = os.path.relpath(fpath, mockups_dir)
                w, h = get_png_dimensions(fpath)
                size = os.path.getsize(fpath)
                lines.append(f"{rel:50s}: {w}x{h} px, {size} bytes")

with open(out_file, "w") as f:
    f.write("\n".join(lines) + "\n")

print("Pure stdlib PNG inspection completed.")
