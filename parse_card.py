import sys

def parse_card(filepath, func_name):
    with open(filepath, 'r') as f:
        lines = f.readlines()

    start_idx = -1
    for i, line in enumerate(lines):
        if func_name in line:
            start_idx = i
            break

    if start_idx == -1:
        return

    brace_count = 0
    in_func = False
    end_idx = -1
    for i in range(start_idx, len(lines)):
        line = lines[i]
        brace_count += line.count('{') - line.count('}')
        if line.count('{') > 0:
            in_func = True
        if in_func and brace_count == 0:
            end_idx = i
            break

    print(f"{start_idx+1}:{end_idx+1}")

if __name__ == '__main__':
    parse_card(sys.argv[1], sys.argv[2])
