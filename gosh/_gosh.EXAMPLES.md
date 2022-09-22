<!-- Created by mkdoc DO NOT EDIT. -->

# Examples

```sh
gosh -pln '"Hello, World!"'
```
This prints Hello, World\!

```sh
gosh -pln 'math.Pi'
```
This prints the value of Pi

```sh
gosh -pln '17*12.5'
```
This prints the results of a simple calculation

```sh
gosh -n -b 'count := 0' -e 'count++' -a-pln 'count'
```
This reads from the standard input and prints the number of lines read

\-n sets up the loop reading from standard input

\-b &apos;count := 0&apos; declares and initialises the counter before the loop

\-e &apos;count\+\+&apos; increments the counter inside the loop

\-a\-pln &apos;count&apos; prints the counter using fmt\.Println after the
loop\.

```sh
gosh -n -b-p '"Radius: "' -e 'r, err := strconv.ParseFloat(_l.Text(), 64)' -e-s iferr -pf '"Area: %9.2f\n", r*r*math.Pi' -p '"Radius: "'
```
This repeatedly prompts the user for a Radius and prints the Area of the
corresponding circle

\-n sets up the loop reading from standard input

\-b\-p &apos;&quot;Radius: &quot;&apos; prints the first prompt before the
loop\.

\-e &apos;r, err := strconv\.ParseFloat\(\_l\.Text\(\), 64\)&apos; sets the
radius from the text read from standard input, ignoring errors\.

\-e\-s iferr checks the error using the &apos;iferr&apos; snippet

\-pf &apos;&quot;Area: %9\.2f\\n&quot;, r\*r\*math\.Pi&apos; calculates and
prints the area using fmt\.Printf\.

\-p &apos;&quot;Radius: &quot;&apos; prints the next prompt\.

```sh
gosh -i -w-pln 'strings.ReplaceAll(string(_l.Text()), "mod/pkg", "mod/v2/pkg")' -- abc.go xyz.go 
```
This changes each line in the two files abc\.go and xyz\.go replacing any
reference to mod/pkg with mod/v2/pkg\. You might find this useful when you are
upgrading a Go module which has changed its major version number\.

The files will be changed and the original contents will be left behind in files
called abc\.go\.orig and xyz\.go\.orig\.

\-i sets up the edit\-in\-place behaviour

\-w\-pln writes to the new, edited copy of the file

```sh
gosh -i -e 'if _fl == 1 {' -w-pln '"// Edited by Gosh!"' -w-pln '' -e '}' -w-pln '_l.Text()' -- abc.go xyz.go 
```
This edits the two files abc\.go and xyz\.go adding a comment at the top of each
file\. It finds the top of the file by checking the built\-in variable \_fl
which gives the line number in the current file

The files will be changed and the original contents will be left behind in files
called abc\.go\.orig and xyz\.go\.orig\.

\-i sets up the edit\-in\-place behaviour

\-w\-pln writes to the new, edited copy of the file

```sh
gosh -http-handler 'http.FileServer(http.Dir("/tmp/xxx"))'
```
This runs a web server that serves files from /tmp/xxx\.

```sh
gosh -web-p '"Gosh!"'
```
This runs a web server \(listening on port 8080\) that returns
&apos;Gosh\!&apos; for every request\.

```sh
gosh -n -e 'if l := len(_l.Text()); l > 80 { ' -pf '"%3d: %s\n", l, _l.Text()' -e '}'
```
This will read from standard input and print out each line that is longer than
80 characters\.

```sh
gosh -snippet-list
```
This will list all the available snippets\.

```sh
gosh -snippet-list -snippet-list-short -snippet-list-part text -snippet-list-constraint iferr
```
This will list just the text of the iferr snippet\.

