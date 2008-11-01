#!/usr/bin/env python
"""
A parallel periodic web fetcher.

Copyright (c) 2008  Dustin Sallings <dustin@spy.net>
"""

import os
import sys
import random

from twisted.web import client
from twisted.internet import defer, protocol, reactor

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
    Download('http://www.google.com/', '/tmp/google.html')().addBoth(
        lambda x: reactor.stop())
    reactor.run()
