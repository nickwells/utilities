<!-- Created by mkdoc DO NOT EDIT. -->

# Examples

```sh
findGoDirs -pkg main
```
This will search recursively down from the current directory for any directory
which contains Go code where the package name is &apos;main&apos;, ignoring the
contents of any \.git directories\. For each directory it finds it will print
the name of the directory\.

```sh
findGoDirs -pkg main -actions install
```
This will install all the Go programs under the current directory\.

```sh
findGoDirs -pkg main -d github.com/nickwells -do install
```
This will install all the Go programs under github\.com/nickwells\.

```sh
findGoDirs -pkg main -not-having .gitignore
```
This will find all the Go directories with code for building commands that
don&apos;t have a \.gitignore  file\. Note that when you run go build in the
directory you will get an executable built in the directory which you don&apos;t
want to check in to git and so you need it to be ignored\.

