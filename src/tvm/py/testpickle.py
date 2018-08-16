
import pickle


class Test(object):
    def __init__(self):
        self.tt = "aa"

def check_base_type(obj):
    base_types = [int, float, str, bool]
    container_types = [dict, list, tuple]
    if type(obj) in base_types:
        return obj
    elif type(obj) in container_types:
        for item in obj:
            if type(obj) is dict:
                value = obj[item]
                if type(item) not in base_types:
                    return None
            else:
                value = item
            if check_base_type(value) is None:
                return None
        return obj
    else:
        return None



if __name__ == '__main__':
    dict1 = {"a": "b", "b": [1, 2, {"c": 3}]}
    obj = check_base_type(dict1)

    print(obj)