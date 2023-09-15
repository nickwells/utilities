package main

import "github.com/nickwells/param.mod/v6/param"

// addExamples this will add examples to the usage message.
func addExamples(ps *param.PSet) error {
	ps.AddExample(`sleepuntil -minute 5 -message "hello"`,
		"This will sleep until the next multiple of 5 minutes and then"+
			" print the message 'hello'"+
			"\n\n"+
			"If you start the program at 09:23 then it will do"+
			" this at 09:25.")
	ps.AddExample(
		`sleepuntil -per-day 6 -do "echo hello >> /tmp/sleepuntil.out" -rc 3`,
		"This will sleep and then append 'hello' to the file"+
			" '/tmp/sleepuntil.out'. It will do this 3 times and then"+
			" exit."+
			"\n\n"+
			"If you start the program at 09:23 then it will do"+
			" this at 12:00, 16:00 and 20:00.")
	ps.AddExample(
		`sleepuntil -second 20 -show-time -r`,
		"This will sleep and then print the time when it wakes up. It"+
			" will do this until you choose to kill the program."+
			"\n\n"+
			"If you start the program at 09:23:51 then it will do"+
			" this at 09:24:00, 09:24:20, 09:24:40, 09:25:00 etc.")
	ps.AddExample(
		`sleepuntil -second 20 -show-time -r -offset -7`,
		"This will sleep and then print the time when it wakes up. It"+
			" will do this until you choose to kill the program."+
			"\n\n"+
			"If you start the program at 09:23:51 then it will do"+
			" this at 09:23:53, 09:24:13, 09:24:33, 09:25:53 etc.")

	return nil
}
