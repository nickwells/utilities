# utilities
miscellaneous useful commands

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
to happen, for instance, every hour, on the hour.

## timeconv
This provides a way of simply converting the time from one locale to another.
