import block

Year = 31536000 #365 * 24 * 60 * 60
Month = 2592000 #30 * 24 * 60 * 60
Hour = 3600 #60 * 60
Week = 604800 #7 * 24 * 60 * 60
Day = 86400 # 24 * 60 * 60
Minute = 60
Second = 1

def time():
    return block.timestamp()


def timedelta(days=0, seconds=0, minutes=0, hours=0, weeks=0):
    assert isinstance(days, int)
    assert isinstance(seconds, int)
    assert isinstance(minutes, int)
    assert isinstance(hours, int)
    assert isinstance(weeks, int)
    s = weeks * Week + days * Day + hours * Hour + minutes * Minute + seconds
    return s


