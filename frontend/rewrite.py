with open("src/views/Layout.vue", "r", encoding="utf-8") as f:
    content = f.read()

script_end = content.find("<template>")
if script_end == -1:
    print("Cannot find <template>")
    exit(1)

script_part = content[:script_end]

style_start = content.rfind("<style")
if style_start == -1:
    print("Cannot find <style")
    exit(1)

style_part = content[style_start:]

import re
with open("patch_layout.py", "r", encoding="utf-8") as f:
    patch = f.read()
    
match = re.search(r'new_template\s*=\s*"""(.*?)"""', patch, re.DOTALL)
if not match:
    print("Cannot find new_template")
    exit(1)

new_template = match.group(1)

with open("src/views/Layout.vue", "w", encoding="utf-8") as f:
    f.write(script_part + new_template + "\n\n" + style_part)
print("Done")
