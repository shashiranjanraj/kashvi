import os
import re

doc_content = open('README.md').read()
pkg_dir = 'pkg'
missing_symbols = {}
all_symbols = {}

for root, dirs, files in os.walk(pkg_dir):
    pkg_name = os.path.basename(root)
    if pkg_name not in all_symbols:
        all_symbols[pkg_name] = []
        
    for file in files:
        if file.endswith('.go') and not file.endswith('_test.go'):
            content = open(os.path.join(root, file)).read()
            # find exported funcs and types
            # Match top-level funcs
            funcs = re.findall(r'^func\s+([A-Z]\w*)\b', content, re.MULTILINE)
            # Match methods: func (r *Receiver) MethodName
            methods = re.findall(r'^func\s+\([^)]+\)\s+([A-Z]\w*)\b', content, re.MULTILINE)
            # Match types
            types = re.findall(r'^type\s+([A-Z]\w*)\b', content, re.MULTILINE)
            
            all_symbols[pkg_name].extend(funcs + methods + types)

for pkg, symbols in all_symbols.items():
    missing = []
    for symbol in set(symbols):
        if symbol not in doc_content:
            missing.append(symbol)
    if missing:
        missing_symbols[pkg] = missing

for pkg, symbols in sorted(missing_symbols.items()):
    print(f"[{pkg}] missing docs for: {', '.join(sorted(symbols))}")
