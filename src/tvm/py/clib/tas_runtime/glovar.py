

import pickle
import signal
import threading

from clib.tas_runtime.block import Block
from clib.tas_runtime.msgxx import Msg
from clib.tas_runtime.storage import Storage
from clib.tas_runtime.address import Address


storage = None
try:
    with open("data.pkl", "rb") as f:
        data = f.read()
        if data == b"":
            storage = Storage()
        else:
            storage = pickle.loads(data)
except FileNotFoundError:
    storage = Storage()
block = Block()
t = threading.Thread(target=block.run)
t.start()
msg = Msg("123", 123, Address("123"))
owner = Address("")


def _exit(signum, frame):
    print("exit")
    block.alive = False
    with open("data.pkl", "wb") as f:
        pickle.dump(storage, f)


signal.signal(signal.SIGINT, _exit)
signal.signal(signal.SIGTERM, _exit)

