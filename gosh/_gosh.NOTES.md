<!-- Created by mkdoc DO NOT EDIT. -->

# Notes

## Gosh \- code sections
The program that gosh will generate is split up into several sections and you
can add code to these sections\. The sections are:



global       \- code at global scope, outside of main

before       \- code at the start of the program

before\-inner \- code before any inner loop

exec         \- code, maybe in a readloop/web handler

after\-inner  \- code after any inner loop

after        \- code at the end of the program



The \.\.\.inner sections are only useful if you have some inner loop \- where
you are looping over a list of files and reading each one\. Otherwise they just
appear immediately before or after their corresponding sections\. before\-inner
appears after before and after\-inner appears before after


## Gosh \- filenames
A list of filenames to be processed can be given at the end of the command line
\(following \-\-\)\. Each filename will be edited to be an absolute path if it
is not already; the current directory will be added at the start of the path\.
If any files are given then some parameter for reading them should be given\.
See the parameters in group: &apos;cmd\-readloop&apos;\.



Note that it is an error if the same file name appears twice\.


## Gosh \- in\-place editing
The files given for editing are checked to make sure that they all exist, that
there is no pre\-existing file with the same name plus the &apos;\.orig&apos;
extension and that there are no duplicate filenames\. If any of these checks
fails the program aborts with an error message\.



If &apos;\-in\-place\-edit&apos; is given then some filenames must be supplied
\(after &apos;\-\-&apos;\)\.



 After you have run this edit program you could use the findCmpRm program to
check that the changes were as expected
### See Parameter
* in\-place\-edit



## Gosh \- shebang scripts
You can use gosh in shebang scripts \(executable files starting with
&apos;\#\!&apos;\)\. Follow the &apos;\#\!&apos; with the full pathname of the
gosh command and the parameter &apos;\-exec\-file&apos; and gosh will construct
your Go program from the contents of the rest of the file and run it\.



The first line should look something like this



\#\!/path/to/gosh \-exec\-file



The rest of the file is Go code to be run inside a main\(\) func\.



Any parameters that you pass to the script will be interpreted as gosh
parameters so you can add extra code to be run\.



You can skip the stage where import statements are populated by passing the
dont\-populate\-imports parameter\. This makes your script run a little faster
and, more importantly, removes the dependency on additional commands \(like
gopls or goimports\)\. If you skip import generation you will need to provide
the packages to be imported through import parameters\.



You might also want to consider setting the full path of the Go command using
the set\-go\-cmd parameter\. This will remove the need for the person running
the shebang script to even have the go command in their path\.
### See Parameters
* after\-file
* before\-file
* dont\-populate\-imports
* exec\-file
* global\-file
* import
* set\-go\-cmd



## Gosh \- snippet comments
Any lines in a snippet file starting with &apos;// snippet:&apos; are not copied
but are treated as comments on the snippet itself\.



A snippet comment can have additional meaning\. If it is followed by one of
these values then the rest of the line is used as described:



\- &apos;note:&apos;

The following text is reported as documentation when the snippets are listed\.

Alternative values are &apos;notes&apos;, &apos;doc&apos; or &apos;docs&apos;



\- &apos;imports:&apos;

The following text is added to the list of import statements\. Note that, by
default, gosh will automatically populate the import statements using a standard
tool\. It runs the first of &apos;gopls imports \-w&apos; or &apos;goimports
\-w&apos; that can be executed\. This should populate the import statements for
you but adding an import comment can ensure that the snippet works even if no
import generator is available\. This also avoids any possible mismatch where the
import populator finds the wrong package\.

An alternative value is &apos;import&apos;



\- &apos;expects:&apos;

Records another snippet that is expected to be given if this snippet is used\.
This allows a chain of snippets to check that all necessary parts have been used
and help to ensure correct usage of the snippet chain\.

This is enforced by the Gosh command\.

Alternative values are &apos;expect&apos; or &apos;comesbefore&apos;



\- &apos;follows:&apos;

Records another snippet that is expected to appear before this snippet is used\.
This allows a chain of snippets to check that the parts have been used in the
right order\.

This is enforced by the Gosh command\.

Alternative values are &apos;follow&apos; or &apos;comesafter&apos;



\- &apos;tag:&apos;

Records a documentary tag\. The text will be split on a &apos;:&apos; and the
first part will be used as a tag with the remainder used as a value\. These are
then reported when the snippets are listed\. These have no semantic meaning and
are purely for documentary purposes\. It allows you to give some structure to
your snippet documentation\.

Suggested tag names might be

   &apos;Author&apos;   to document the snippet author

   &apos;Env&apos;      for an environment variable the snippet uses

   &apos;Declares&apos; for a variable that it declares\.

An alternative value is &apos;tags&apos;
### See Note
* Gosh \- snippets



## Gosh \- snippet directories
By default snippets will be searched for in standard directories\.



The directories are searched in the order given above and the first file
matching the name of the snippet will be used\. Any extra directories, since
they are added at the start of the list, will be searched before the default
ones\.
### See Parameters
* snippet\-list\-dir
* snippets\-dir

### See Note
* Gosh \- snippets



## Gosh \- snippets
You can introduce pre\-defined blocks of code \(called snippets\) into your
script\. gosh will search through a list of directories for a file with the
snippet name and insert that into your script\. A filename with a full path can
also be given\. Any inserted code is prefixed with a comment showing which file
it came from to help with debugging\.



A suggested standard is to name any variables that you declare in a snippet file
with a leading double underscore\. This will ensure that the names neither clash
with any gosh\-declared variables nor any variables declared by the user\.



It is also suggested that sets of snippets which must be used together should be
grouped into their own sub\-directory in the snippets directory and named with
leading digits to indicate the order that they must be applied\.
### See Note
* Gosh \- snippet directories



## Gosh \- variables
gosh will create some variables as it builds the program\. These are all listed
below\. You should avoid creating any variables yourself with the same names and
you should not change the values of any of these\. Note that they all start with
a single underscore so provided you start all your variable names with a letter
\(as usual\) you will not clash\.



\_arg  string               the current argument

\_args \[\]string             the list of arguments

\_err  error                an error

\_f    \*os\.File             the file being read

\_fl   int                  the current line number in the file

\_fn   string               the name of the file \(or stdin\)

\_fns  \[\]string             the list of names of the files

\_l    \*bufio\.Scanner       a buffered scanner used to read the files

\_lp   \[\]string             the parts of the line \(when split\)

\_req  \*http\.Request        the request to the web server

\_rw   http\.ResponseWriter  the response writer for the web server

\_sre  \*regexp\.Regexp       the regexp used to split lines

\_w    \*os\.File             the file written to if editing in place


