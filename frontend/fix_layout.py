import re

with open("src/views/Layout.vue", "r", encoding="utf-8") as f:
    content = f.read()

# get template from patch_layout.py
with open("patch_layout.py", "r", encoding="utf-8") as f:
    patch_script = f.read()

# extract new_template string
import ast
match = re.search(r'new_template\s*=\s*"""(.*?)"""', patch_script, re.DOTALL)
if match:
    new_template = match.group(1)
    
    # Remove existing template
    content = re.sub(r'<template>.*?</template>', '', content, flags=re.DOTALL)
    
    # insert new template before <style scoped>
    content = content.replace('<style scoped>', new_template + '\n\n<style scoped>')
    
    with open("src/views/Layout.vue", "w", encoding="utf-8") as f:
        f.write(content)
        print("Successfully updated Layout.vue")
else:
    print("Failed to find new_template")
