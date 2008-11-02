== pfetch - A Parallel Periodic URL Fetcher Utility ==

This is a really simple twisted app, but it solves a real problem I had, so
maybe it'll help you, too.

I've got a bunch of various URLs from which I pull down content regularly via
cron to do some processing.  Having a full-time process fetching all of them
without worrying about cron job overlaps, network timeouts slowing things down,
etc... is going to make my machine a lot happier.
