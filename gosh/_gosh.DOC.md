<!-- Created by mkdoc DO NOT EDIT. -->

![gosh logo](_images/gosh.png)
# gosh

This allows you to write lines of Go code and have them run for you in a
framework that provides the main\(\) func and any necessary boilerplate code for
some common requirements\. The resulting program can be preserved for subsequent
editing\.

You can run the code in a loop that will read lines from the standard input or
from a list of files and, optionally, split each line into fields\.

Alternatively you can quickly generate a simple webserver\.

It&apos;s faster than opening an editor and writing a Go program from scratch
especially if there are only a few lines of non\-boilerplate code\. You can also
save the program that it generates and edit that if the few lines become many
lines\. The workflow would be that you use this to make the first few iterations
of the command and if that is sufficient then just stop\. If you need to do more
then save the file and edit it just like a regular Go program\.



## Parameters

This uses the `param` package and so it has access to the help parameters
which give a comprehensive message describing the usage of the program and
the parameters you can give. The `-help` parameter on its own will print the
standard parameters that the program can accept but you can also give
parameters to show both more or less help, in more or less detail. Other
standard parameters allow you to explore where parameters have been set and
where they can be set. The description of the `-help` parameter is a good
place to start to explore the help available.

The intention of the `param` package is to provide complete documentation
for the program from the command line.
## Installation

To install gosh you should run the following command:

``` sh
go install github.com/nickwells/utilities/gosh@latest
```

This will build `gosh` and install it into your `GOPATH bin` directory
(by default `$HOME/go/bin`).

## Setup

The first thing you'll want to do is to glance at the manual. The `gosh`
command has built-in documentation; just call it with the `-help` parameter
(or the `-help-full` parameter for the complete manual).

You don't need to do anything else to use; `gosh` is a single self-contained
binary but if you use `zsh` and want to set up parameter completion, `gosh`
can do this; see the help message for the appropriate parameters.

Similarly, you might want to set up the default snippets; see the section
below for details.

## Snippets

You can insert the contents of files into your script. These files are called
snippets and can be kept in standard directories where you don't need to give
the whole pathname.

There are some sample snippets in the `_snippets` directory which can be
found in the repository `github.com/nickwells/utilities` under the `gosh`
subdirectory.

You can copy them into a snippets directory with the following
command (on Linux):

``` sh
cp -r --suffix=.orig --backup ./_snippets/* <your-snippets-dir>
```

Choose one of the available snippets directories to replace the placeholder
`<your-snippets-dir>` above. These directories can be listed by calling
`gosh` with the `-snippet-list-dir` parameter.

If you aleady have snippets in your snippets directories the `cp` command
given above will preserve any that have a clash of names with those given in
the `_snippets` directory and you can use the `findCmpRm` command to examine
them and tidy up.

## Printing

Printing is a common requirement of a script and so `gosh` offers some
parameters to make it easier to add print statements to your script. The
following two scripts are equivalent:

``` sh
gosh -e 'fmt.Print("Hello, World!")'
```

and

``` sh
gosh -p '"Hello, World!"'
```

There are several variants, `-pln` uses `fmt.Println` and `-pf` uses
`fmt.Printf`. There are also variants allowing you to print in different
sections of the script, see the help pages for details.


## Examples
For examples [see here](_gosh.EXAMPLES.md)


## See Also
For external references [see here](_gosh.REFERENCES.md)


## Notes
For additional notes [see here](_gosh.NOTES.md)
