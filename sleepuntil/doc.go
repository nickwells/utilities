/*
The sleepuntil program will sleep up to a specified time and then either exit
or else perform some specified action. The time to sleep until can either be
given explicitly or else can be given in terms of a regular interval. If a
regular interval is given then the first wait will be until the first
instance of that interval. For instance, if you have specified that the
program should wake up every hour on the hour and it is five minutes to the
hour when you run the program then the first sleep will only be for 5 minutes
not for an hour.
*/
package main
