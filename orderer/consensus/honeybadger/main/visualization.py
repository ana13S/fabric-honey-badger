import sys

TRANSACTION_FILE = sys.argv[1]
BLOCK_WIDTH = 50
ROWS_IN_ONE_BLOCK = 3
COLUMNS_LEFT_TO_BLOCK = 5

ARROW = [
    " /\\",
    "/||\\",
    " ||  ",
]

BLOCK_START = "-" * BLOCK_WIDTH
BLOCK_ROW = "|" + " " * (BLOCK_WIDTH - 2) + "|"
BLOCK_END = "-" * BLOCK_WIDTH

BLOCK = [BLOCK_START] + [BLOCK_ROW] * ROWS_IN_ONE_BLOCK + [BLOCK_END]
START_BLOCK = [BLOCK_START] + [BLOCK_ROW] * 2 + [BLOCK_END]

def PRINT_ARROW():
    for l in ARROW:
        print(" " * (int(BLOCK_WIDTH / 2) + COLUMNS_LEFT_TO_BLOCK) + l)

def PRINT_BLOCK(blk, msgs):
    assert len(msgs) <= ROWS_IN_ONE_BLOCK
    for i, l in enumerate(blk):
        if i == 0 or i == len(blk) - 1:
            print (" " * COLUMNS_LEFT_TO_BLOCK + l)
        else:
            if len(msgs) >= i:
                msg = msgs[i - 1]
                msg_length = len(msg)
                left_space = int((BLOCK_WIDTH - 2 - msg_length) / 2)
                right_space =  BLOCK_WIDTH - 2 - msg_length - left_space
                print (" " * COLUMNS_LEFT_TO_BLOCK + "|" + " " * left_space + msg + " " * right_space + "|")
            else:

                print (" " * COLUMNS_LEFT_TO_BLOCK + l)

# n starts from 1
def find_nth_block(n):
    separator_between_blocks_seen = 0
    transactions = []
    with open(TRANSACTION_FILE) as f:
        for l in f.readlines():
            if l.strip() == "":
                separator_between_blocks_seen += 1
                if (separator_between_blocks_seen == n):
                    # we have found the transactions for the nth block.
                        return transactions
                else:
                    transactions = []
            else:
                transactions.append(l.strip())
    # Not found nth block.
    return []


PRINT_BLOCK(START_BLOCK, ["START_OF_CHAIN"])
i = 1
while True:
    msgs = find_nth_block(i)
    if (msgs == []):
        continue
    else:
        PRINT_ARROW()
        PRINT_BLOCK(BLOCK, msgs)
        i += 1

# PRINT_BLOCK(START_BLOCK, ["START_OF_CHAIN"])
# PRINT_ARROW()
# PRINT_BLOCK(BLOCK, ["A sends 100 to B", "B sends 200 to C", "G send F 300"])
# PRINT_ARROW()
# PRINT_BLOCK(BLOCK, ["A wins lottery", "A gives lottery to B", "C steals lottery"])
# PRINT_ARROW()
# PRINT_BLOCK(BLOCK, ["A sends 100 to B", "B sends 200 to C", "G send F 300"])
# PRINT_ARROW()
# PRINT_BLOCK(BLOCK, ["A wins lottery", "A gives lottery to B", "C steals lottery"])
# PRINT_ARROW()
# PRINT_BLOCK(BLOCK, ["A sends 100 to B", "B sends 200 to C", "G send F 300"])
# PRINT_ARROW()
# PRINT_BLOCK(BLOCK, ["A wins lottery", "A gives lottery to B", "C steals lottery"])
# PRINT_ARROW()
# PRINT_BLOCK(BLOCK, ["A sends 100 to B", "B sends 200 to C", "G send F 300"])
# PRINT_ARROW()
# PRINT_BLOCK(BLOCK, ["A wins lottery", "A gives lottery to B", "C steals lottery"])
# PRINT_ARROW()
# PRINT_BLOCK(BLOCK, ["A sends 100 to B", "B sends 200 to C", "G send F 300"])
# PRINT_ARROW()
# PRINT_BLOCK(BLOCK, ["A wins lottery", "A gives lottery to B", "C steals lottery"])
# PRINT_ARROW()
# PRINT_BLOCK(BLOCK, ["A sends 100 to B", "B sends 200 to C", "G send F 300"])
# PRINT_ARROW()
# PRINT_BLOCK(BLOCK, ["A wins lottery", "A gives lottery to B", "C steals lottery"])
# PRINT_ARROW()
# PRINT_BLOCK(BLOCK, ["A sends 100 to B", "B sends 200 to C", "G send F 300"])
# PRINT_ARROW()
# PRINT_BLOCK(BLOCK, ["A wins lottery", "A gives lottery to B", "C steals lottery"])
# PRINT_ARROW()
# PRINT_BLOCK(BLOCK, ["A sends 100 to B", "B sends 200 to C", "G send F 300"])
# PRINT_ARROW()
# PRINT_BLOCK(BLOCK, ["A wins lottery", "A gives lottery to B", "C steals lottery"])
# PRINT_ARROW()
# PRINT_BLOCK(BLOCK, ["A sends 100 to B", "B sends 200 to C", "G send F 300"])
# PRINT_ARROW()
# PRINT_BLOCK(BLOCK, ["A wins lottery", "A gives lottery to B", "C steals lottery"])