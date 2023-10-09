## Installation

To install gosh you should run the following command:

``` sh
go install github.com/nickwells/utilities/gosh@latest
```

This will build `gosh` and install it into your `GOPATH bin` directory
(by default `$HOME/go/bin`).

You should then check the installation. Run the following command:

``` sh
gosh -pre-check
```

This will check that various tools that gosh uses are installed and will give
advice on how to install them if they are missing. It will also check that
the standard snippets have been installed and, again, it will give advice on
how to install them if not.

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
found in the repository `github.com/nickwells/utilities` under the
`gosh.snippet` subdirectory.

The recommended way to install these standard snippets is to use the
`gosh.snippet` command. Alternatively, you can copy them from the source code
into a snippets directory with the following command (on Linux):

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
