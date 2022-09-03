/*
gosh will take go code entered as a command line parameter and wrap it in a
main() function. Then it will use gofmt (or goimports if it is installed) to
format the code and lastly call go run to compile and run the code.
*/
package main
