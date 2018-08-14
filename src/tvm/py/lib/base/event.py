class Event(object):
    @staticmethod
    def emit(event_name, *param):
        print("Event: ", event_name, param)
