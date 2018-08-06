

import os


a=10


g = {}
l = {}
print(exec("a = a + 1",g,l))

print(exec("a = a + 1",g,l))

print(g)
print(l)


print(a)

