import pickle
import signal
import threading

from lib.block import Block
from lib.msg import Msg
from lib.storage import Storage

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
msg = Msg("123", "123", "123")


def _exit(signum, frame):
    print("exit")
    block.alive = False
    with open("data.pkl", "wb") as f:
        pickle.dump(storage, f)


signal.signal(signal.SIGINT, _exit)
signal.signal(signal.SIGTERM, _exit)