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
certain pattern is matched and to skip otherwise matching lines if they match a
pattern



You can add these additional features using the &apos;check&apos; parameter\.
### See Parameters
* check
* having\-build\-tag
* having\-go\-generate



