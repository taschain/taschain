import threading

from lib.block import Block
from lib.msg import Msg
from lib.storage import Storage

storage = Storage()
block = Block()
t = threading.Thread(target=block.run)
t.start()
msg = Msg("123", "123", "123")
