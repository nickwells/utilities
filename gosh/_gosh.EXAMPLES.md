<!-- Created by mkdoc DO NOT EDIT. -->

# Examples

```sh
gosh -pln '"Hello, World!"'
```
This prints Hello, World!

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

-n sets up the loop reading from standard input

-b 'count := 0' declares and initialises the counter before the loop

-e 'count++' increments the counter inside the loop

-a-pln 'count' prints the counter using fmt.Println after the loop.

```sh
gosh -n -b-p '"Radius: "' -e 'r, _ := strconv.ParseFloat(_l.Text(), 10)' -pf '"Area: %9.2f\n", r*r*math.Pi' -p '"Radius: "'
```
This repeatedly prompts the user for a Radius and prints the Area of the
corresponding circle

-n sets up the loop reading from standard input

-b-p '"Radius: "' prints the first prompt before the loop.

-e 'r, _ := strconv.ParseFloat(_l.Text(), 10)' sets the radius from the text
read from standard input, ignoring errors.

-pf '"Area: %9.2f\n", r*r*math.Pi' calculates and prints the area using
fmt.Printf.

-p '"Radius: "' prints the next prompt.

```sh
gosh -i -w-pln 'strings.ReplaceAll(string(_l.Text()), "mod/pkg", "mod/v2/pkg")' -- abc.go xyz.go 
```
This changes each line in the two files abc.go and xyz.go replacing any
reference to mod/pkg with mod/v2/pkg. You might find this useful when you are
upgrading a Go module which has changed its major version number. The files will
be changed and the original contents will be left behind in files called
abc.go.orig and xyz.go.orig.

```sh
gosh -http-handler 'http.FileServer(http.Dir("/tmp/xxx"))'
```
This runs a web server that serves files from /tmp/xxx.

```sh
gosh -web-p '"Gosh!"'
```
This runs a web server (listening on port 8080) that returns 'Gosh!' for every
request.

```sh
gosh -n -e 'if l := len(_l.Text()); l > 80 { ' -pf '"%3d: %s\n", l, _l.Text()' -e '}'
```
This will read from standard input and print out each line that is longer than
80 characters.

