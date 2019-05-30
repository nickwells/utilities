# utilities
Miscellaneous useful commands.

All these tools use the standard param package to handle command-line flags
and so they support the standard '-help' parameter which will print out a
comprehensive usage message.

## statfs
This provides an equivalent to the `df` command but in a form that is easier
to use in a shell script. The default output is easy for a human to
understand but with the right flags set it can deliver just the value
required.

## sleepuntil
This provides a way of repeatedly sleeping until a particular time is
reached. This time can be given as a particular time and date or as a
specification of some fragment of the day. It will sleep until that fragment
of the day is reached. For instance if you choose to sleep until hour 8 then
it will sleep until 8:00 or 16:00 or midnight. You can specify if you want it
to repeat indefinitely or for a set number of times and you can specify what
you want to happen when it wakes up. This can be useful if you want something
to happen, for instance, every hour, on the hour, but only within a script
(otherwise you could use cron).

## timeconv
This provides a way of simply converting the time from one locale to
another. This can be useful when you are working with colleagues in other
timezones with different daylight-saving rules.

## retirement
This will perform a very basic financial analysis of a retirement
portfolio. It does a very simple modelling of the performance of a retirement
portfolio over a number of years and allows you to model different scenarios.

## bankAcAnalysis
This will read a csv-file holding bank transactions and will try to group
them according to various rules you have supplied. This is still very much a
work in progress but is of some use which is why it is here.

## mkparamfilefunc
this is intended to be used with go generate to construct functions that can
be used to set the parameter files for packages and commands.
