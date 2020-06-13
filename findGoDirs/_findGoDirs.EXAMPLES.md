<!-- Created by mkdoc DO NOT EDIT. -->

# Examples

```sh
findGoDirs -pkg main
```
This will search recursively down from the current directory for any directory
which contains Go code where the package name is 'main', ignoring the contents
of any .git directories. For each directory it finds it will print the name of
the directory.

```sh
findGoDirs -pkg main -actions install
```
This will install all the Go programs under the current directory.

```sh
findGoDirs -pkg main -d github.com/nickwells -do install
```
This will install all the Go programs under github.com/nickwells.

