import sys
sys.path.append("lib")

import os
import random

from twisted.application import service
from twisted.web import microdom
from twisted.internet import task, reactor

import pfetch

application = service.Application("pfetch")

r=random.Random()
doc=microdom.parse('urls.xml')

for u in doc.getElementsByTagName("url"):
    a=u.attributes
    freq=int(a['freq'])
    lc=task.LoopingCall(pfetch.Download(a['href'], a['output']))
    reactor.callLater(r.randint(0, freq), lc.start, freq)
