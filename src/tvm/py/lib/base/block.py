import hashlib
import random
import time


class Block(object):
    def __init__(self):
        self.blocks = []
        self.height = 0
        self.sha256 = hashlib.sha256()

    def number(self):
        return self.height

    def blockhash(self, height):
        if height < len(self.blocks):
            return self.blocks[height + 1]

    def run(self):
        while True:
            self.height += 1
            time.sleep(10)
            self.sha256.update(bytes(self.height))
            res = self.sha256.hexdigest()
            self.blocks.append(res)

    def difficulty(self):
        return random.randint(1, 10)

    def coinbase(self):
        sha256 = hashlib.sha256()
        r = random.randint(1, 100000000)
        sha256.update(bytes(r))
        return sha256.hexdigest()

    def gaslimit(self):
        return random.randint(1000, 100000)

    def timestamp(self):
        return time.time()