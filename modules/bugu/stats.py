import re
from collections import Counter

type_counts = Counter()
lang_counts = Counter()

with open("/Users/kael/Projects/Finnegan/bugu/radar_targets.yml", "r") as f:
    text = f.read()

types = re.findall(r"type:\s*(site|engine|footprint)", text)
for t in types:
    type_counts[t] += 1

languages = re.findall(r"languages:\s*\[(.*?)\]", text)
for line in languages:
    langs = [l.strip().strip('\"').strip('\'') for l in line.split(",")]
    for lang in langs:
        if lang:
            lang_counts[lang] += 1

print(f"Total Targets: {sum(type_counts.values())}")
print("\n[ Breakdown by Type ]")
for k, v in type_counts.most_common():
    print(f" - {k.capitalize()}: {v}")

print("\n[ Breakdown by Language ]")
for k, v in lang_counts.most_common():
    print(f" - {k}: {v}")
