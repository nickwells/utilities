## How to get gosh

To install gosh you should run the following command:

```
go install github.com/nickwells/utilities/gosh@latest
```

This will build `gosh` and install it into your `GOPATH bin` directory
(by default `$HOME/go/bin`).

## Snippets

You can insert the contents of files into your script. These files are called
snippets and can be kept in standard directories where you don't need to give
the whole pathname.

There are some sample snippets in the `_snippets` directory.

You can copy them into a snippets directory with the following
command (on Linux):

```
cp -r --suffix=.orig --backup ./_snippets/* <your-snippets-dir>
```

Choose one of the available snippets directories to replace the placeholder
`<your-snippets-dir>` above. These directories can be listed by passing
the `snippet-list-dir` parameter.

If you aleady have snippets in your snippets directories the `cp` command
given above will preserve any that have a clash of names with those given in
the `_snippets` directory and you can use the `findCmpRm` command to examine
them and tidy up.

## Printing

Printing is a common requirement of a script and so `gosh` offers some
parameters to make it easier to add print statements to your script. The
following two scripts are equivalent:

```
gosh -e 'fmt.Print("Hello, World!")'
```

and

```
gosh -p '"Hello, World!"'
```

There are several variants, `-pln` uses `fmt.Println` and `-pf` uses
`fmt.Printf`. There are also variants allowing you to print in different
sections of the script, see the help pages for details.
