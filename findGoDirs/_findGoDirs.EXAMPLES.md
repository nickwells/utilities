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

```sh
findGoDirs -having-go-generate
```
This will find all the Go directories with go:generate comments\. These are the
directories where you might need to run &apos;go generate&apos; or where
&apos;go generate&apos; might have changed the directory contents\.

```sh
findGoDirs -having-go-generate -do content
```
This will find all the Go directories with go:generate comments and prints the
matching lines\.

```sh
findGoDirs -having-content 'nolint=//nolint:' -do content
```
This will find all the Go directories with some file having a nolint comment and
prints the matching lines\.

```sh
findGoDirs -having-content 'nolint=//nolint:' -having-content 'nolint.skip=errcheck' -do content
```
This will find all the Go directories with some file having a nolint comment but
where the line matching //nolint doesn&apos;t also match errcheck and prints the
matching lines\.

