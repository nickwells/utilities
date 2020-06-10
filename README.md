# utilities
Miscellaneous useful commands.

All these tools use the standard param package to handle command-line flags
and so they support the standard '-help' parameter which will print out a
comprehensive usage message.

## gosh
This is a tool for running Go code from the command line.
[See here](gosh/_gosh.DOC.md).

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

## mkparamfilefunc
This is intended to be used with go generate to construct functions that can
be used to set the parameter files for packages and commands. It will write a
Go file with functions that can be passed to a call to paramset.NewOrDie to
set the per-command config files. This will allow the user of a program to
set parameters that they want to use every time the program is run.

## mkpkgerr
This will generate the code to provide a package-specific error type
(pkgError) which allows errors from your package to be distinguished from
errors from other sources. It defines an interface called Error which will be
satisfied only by errors from your package. The pkgError is not exported and
so cannot be used outside of the package but does satisfy the
package-specific Error interface (and also the standard error interface). It
also provides a local pkgErrorf function that can be used to generate a
pkgError. The pkgError is a renaming of string and so a string can simply be
cast to a pkgError.

## mkdoc
This is a tool for generating markdown files documenting programs.
[See here](mkdoc/_mkdoc.DOC.md).

## findCmpRm
This finds all files in a given directory with a given extension and compares
them against corresponding files without the extension. Then the user is
prompted to delete the file with the extension. The command name echoes this:
find, compare, remove.

It is most useful in conjunction with the testhelper package. The testhelper
package will retain the original contents of a golden file in a file of the
same name with an extension of '.orig'. This command will help you to review
the changes and tidy up afterwards.
