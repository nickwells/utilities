<!-- Created by mkdoc DO NOT EDIT. -->

# Notes

## Content Checks
You can constrain the Go directories this command will find by checking that a
matching directory has at least one file containing certain content\.



This feature can by useful, for instance, to find directories having files with
go:generate comments so you know if you need to run &apos;go generate&apos; in
them\.



There are some common searches which have dedicated parameters for setting them:
&apos;having\-build\-tag&apos; and &apos;having\-go\-generate&apos;\. These have
all the correct patterns preset and it is recommended that you use these\.



A content checker has at least a pattern for matching lines but it can be
extended to only check files matching a pattern, to stop matching after a
sertain pattern is matched and to skip otherwise matching lines if they match an
additional pattern



You can add these additional features using the &apos;having\-content&apos;
parameter\. You repeat the checker name and add

    a period \(&apos;\.&apos;\),

    a part name,

    an equals \(&apos;=&apos;\)

    and the pattern for that part\.

Valid part names are:

filename, skip, stop



Before you can add a part you must first create the checker by giving a checker
name and the match pattern \(no &apos;\.part&apos; is needed\)
### See Parameters
* having\-build\-tag
* having\-content
* having\-go\-generate



