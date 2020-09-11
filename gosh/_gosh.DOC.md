<!-- Created by mkdoc DO NOT EDIT. -->

# gosh

This allows you to write lines of Go code and have them run for you in a
framework that provides the main() func and any necessary boilerplate code for
some common requirements. The resulting program can be preserved for subsequent
editing.

You can run the code in a loop that will read lines from the standard input or
from a list of files and, optionally, split each line into fields.

Alternatively you can quickly generate a simple webserver.

It's faster than opening an editor and writing a Go program from scratch
especially if there are only a few lines of non-boilerplate code. You can also
save the program that it generates and edit that if the few lines become many
lines. The workflow would be that you use this to make the first few iterations
of the command and if that is sufficient then just stop. If you need to do more
then save the file and edit it just like a regular Go program.



There are some sample snippets in the `_snippets` directory.

You can copy them into a snippets directory with the following
command (on Linux):

```
cp -r --suffix=.orig --backup ./_snippets/* <your-snippets-dir>
```

Choose one of the available snippets directories to replace the placeholder
`<your-snippets-dir>` above. These directories are listed in the program
notes:

```
gosh -help-show notes
```

If you aleady have snippets in your snippets directories the `cp` command
given above will preserve any that have a clash of names with those given in
the `_snippets` directory and you can use the `findCmpRm` command to examine
them and tidy up.


## Examples
For examples [see here](_gosh.EXAMPLES.md)


## See Also
For external references [see here](_gosh.REFERENCES.md)
