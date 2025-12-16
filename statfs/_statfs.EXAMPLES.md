<!-- Created by mkdoc DO NOT EDIT. -->

# Examples

```sh
statfs
```
This will print the directory being tested \(the current directory by default\)
and the available space in bytes\.

```sh
statfs -show avail -no-label -units GB -- /home/me
```
This will print just the available space \(in Gigabytes\) without any label\.
The filesystem to be reported on is the one on which /home/me is found\.

This form is useful if you want to use the result in a shell script since you
don&apos;t need to pass the output to any other programs to strip any labels\.

