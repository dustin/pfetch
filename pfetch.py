#!/usr/bin/env python
"""
A parallel periodic web fetcher.

Copyright (c) 2008  Dustin Sallings <dustin@spy.net>
"""

import os

from twisted.internet import reactor
from twisted.web import client

class ProcessHandler(object):

    def __init__(self, u):
        self.url=u

    def makeConnection(self, process):
        print "Starting job for %s" % self.url

    def childDataReceived(self, childFd, data):
        print "Heard from child on fd#%d: %s" % (childFd, data)

    def childConnectionLost(self, childFd):
        pass

    def processEnded(self, reason):
        print "Finished job for %s" % self.url

class Download(object):

    def __init__(self, url, file, cmd, args):
        self.url=url
        self.file=file
        self.tmpfile = file + ".tmp"
        self.cmd=cmd
        self.args=args

    def __onComplete(self, v):
        os.rename(self.tmpfile, self.file)
        if self.cmd:
            e={'PFETCH_URL': self.url, 'PFETCH_FILE': self.file}
            args=[self.cmd] + self.args
            reactor.spawnProcess(ProcessHandler(self.url), self.cmd, args, e)

    def __call__(self):
        return client.downloadPage(self.url, self.tmpfile).addCallback(
            self.__onComplete)
