<!-- Created by mkdoc DO NOT EDIT. -->

# gosh

This allows you to write lines of Go code and have them run for you in a
framework that provides the main() func and any necessary boilerplate code for
some common requirements. The resulting program can be preserved for subsequent
editing.

You can run the code in a loop that will read lines from the standard input or
from a list of files and, optionally, split them into fields.

Alternatively you can quickly generate a simple webserver.

It's faster than opening an editor and writing a Go program from scratch
especially if there are only a few lines of non-boilerplate code. You can also
save the program that it generates and edit that if the few lines become many
lines. The workflow would be that you use this to make the first few iterations
of the command and if that is sufficient then just stop. If you need to do more
then save the file and edit it just like a regular Go program.

By default the program will be generated in a temporary directory and executed
from there so that any paths used should be given in full rather than relative
to your current directory.



## Examples
For examples [see here](_gosh.EXAMPLES.md)


## See Also
For external references [see here](_gosh.REFERENCES.md)
