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

class DownloadFactory(client.HTTPDownloader):

    def gotHeaders(self, headers):
        # Super call...
        client.HTTPDownloader.gotHeaders(self, headers)
        self.headers=headers

    def pageEnd(self):
        if not self.file:
            return
        try:
            self.file.close()
        except IOError:
            self.deferred.errback(failure.Failure())
            return
        self.deferred.callback((self.headers, self.value))

def myDownloadPage(url, file, *args, **kwargs):
    scheme, host, port, path = client._parse(url)
    factory = DownloadFactory(url, file, *args, **kwargs)
    if scheme == 'https':
        from twisted.internet import ssl
        contextFactory = ssl.ClientContextFactory()
        reactor.connectSSL(host, port, factory, contextFactory)
    else:
        reactor.connectTCP(host, port, factory)
    return factory.deferred

class Download(object):

    def __init__(self, url, file, cmd, args):
        self.url=url
        self.file=file
        self.tmpfile = file + ".tmp"
        self.cmd=cmd
        self.args=args
        self.etag=None

    def __saveEtag(self, headers):
        self.etag = headers.get('etag', [None])[0]

    def __onComplete(self, v):
        headers, val = v
        self.__saveEtag(headers)
        os.rename(self.tmpfile, self.file)
        if self.cmd:
            e={'PFETCH_URL': self.url, 'PFETCH_FILE': self.file}
            args=[self.cmd] + self.args
            reactor.spawnProcess(ProcessHandler(self.url), self.cmd, args, e)

    def __call__(self):
        headers = {}
        if self.etag:
            headers['If-None-Match'] = self.etag
        def p(v):
            print "Error on %s: %s" % (self.url, str(v))
        return myDownloadPage(self.url, self.tmpfile, headers=headers
            ).addCallback(self.__onComplete).addErrback(p)
