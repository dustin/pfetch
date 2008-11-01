#!/usr/bin/env python
"""
A parallel periodic web fetcher.

Copyright (c) 2008  Dustin Sallings <dustin@spy.net>
"""

import os
import sys
import random

from twisted.web import client, microdom
from twisted.internet import defer, task, protocol, reactor

class Download(object):

    def __init__(self, url, file):
        self.url=url
        self.file=file
        self.tmpfile = file + ".tmp"

    def __onComplete(self, v):
        print "Completed", self.url
        os.rename(self.tmpfile, self.file)

    def __call__(self):
        print "Fetching", self.url
        return client.downloadPage(self.url, self.tmpfile).addCallback(
            self.__onComplete)

if __name__ == '__main__':
    r=random.Random()
    doc=microdom.parse(sys.argv[1])

    for u in doc.getElementsByTagName("url"):
        a=u.attributes
        freq=int(a['freq'])
        lc=task.LoopingCall(Download(a['href'], a['output']))
        reactor.callLater(r.randint(0, freq), lc.start, freq)

    reactor.run()
