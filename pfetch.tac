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

def find_command(u):
    cmd=None
    args=[]

    cmdNodes=u.getElementsByTagName("command")
    if len(cmdNodes):
        cmd=cmdNodes[0].attributes["path"]
        for a in cmdNodes[0].getElementsByTagName("arg"):
            args.append(a.firstChild().data)

    return cmd, args

for u in doc.getElementsByTagName("url"):
    cmd, args = find_command(u)
    a=u.attributes
    print "got", cmd, `args`, "for", a['href']
    freq=int(a['freq'])
    lc=task.LoopingCall(pfetch.Download(a['href'], a['output'], cmd, args))
    reactor.callLater(r.randint(0, freq), lc.start, freq)
