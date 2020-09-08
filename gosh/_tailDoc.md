There are some sample snippets in the `_snippets` directory.

You can copy them into a snippets directory with the following
command (on Linux):

```
cp -r --suffix=.orig --backup ./_snippets ...snippets
```

The available snippets directories are listed in the program notes:

```
gosh -help-show notes
```

If you aleady have snippets in your snippets directories the `cp` command
given above will preserve any that have a clash of names with those given in
the `_snippets` directory and you can use the `findCmpRm` command to examine
them and tidy up.
