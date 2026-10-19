import sys

def get_block(filepath, start_pattern, end_pattern):
    with open(filepath, 'r') as f:
        lines = f.readlines()

    start_idx = -1
    for i, line in enumerate(lines):
        if start_pattern in line:
            start_idx = i
            break

    if start_idx == -1:
        print(f"Start pattern '{start_pattern}' not found")
        return

    end_idx = -1
    for i in range(start_idx+1, len(lines)):
        if end_pattern in line:
            end_idx = i
            break

    if end_idx == -1:
        print(f"End pattern '{end_pattern}' not found")
        return

    print(f"{start_idx+1}:{end_idx+1}")

if __name__ == '__main__':
    get_block(sys.argv[1], sys.argv[2], sys.argv[3])
