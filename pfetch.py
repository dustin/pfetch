#!/usr/bin/env python
"""
A parallel periodic web fetcher.

Copyright (c) 2008  Dustin Sallings <dustin@spy.net>
"""

import os

from twisted.web import client

class Download(object):

    def __init__(self, url, file):
        self.url=url
        self.file=file
        self.tmpfile = file + ".tmp"

    def __onComplete(self, v):
        os.rename(self.tmpfile, self.file)

    def __call__(self):
        return client.downloadPage(self.url, self.tmpfile).addCallback(
            self.__onComplete)
