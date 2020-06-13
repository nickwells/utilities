<!-- Created by mkdoc DO NOT EDIT. -->

# Examples

```sh
findCmpRm -diff sdiff -diff-args '-w,170'
```
This will use sdiff to compare the files rather than the default program (diff)

```sh
findCmpRm -diff-args '-W,170,-y,--color=always' -less-args=-R
```
This will use show the differences in two columns, side by side, with
differences highlighted in colour and with less taking the colour output and
displaying it.

You might want to put these parameters in the configuration file so that you
don't have to repeatedly set them on each use of the program.

```sh
findCmpRm -d testdata -r
```
This will search the testdata directory and any subdirectories for the files to
process.

Any directory called 'testdata' is ignored by the go tools as are directories
with names starting with an '_'

