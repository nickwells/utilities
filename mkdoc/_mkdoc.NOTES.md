<!-- Created by mkdoc DO NOT EDIT. -->

# Notes

## Files generated
Each of the generated Markdown files will have a name starting with an
underscore followed by the name of the program itself\. The files to be
generated are as follows:



The text from the &quot;intro&quot; section of the help message is written to a
file ending &quot;\.DOC\.md&quot;\. Text to come before this is in a file called
&quot;\_headDoc\.md&quot; and any text to come after in a file called
&quot;\_tailDoc\.md&quot;\.



The text from the &quot;examples&quot; section of the help message is written to
a file ending &quot;\.EXAMPLES\.md&quot;\. Text to come before this is in a file
called &quot;\_headExamples\.md&quot; and any text to come after in a file
called &quot;\_tailExamples\.md&quot;\.



The text from the &quot;refs&quot; section of the help message is written to a
file ending &quot;\.REFERENCES\.md&quot;\. Text to come before this is in a file
called &quot;\_headReferences\.md&quot; and any text to come after in a file
called &quot;\_tailReferences\.md&quot;\.



The text from the &quot;notes&quot; section of the help message is written to a
file ending &quot;\.NOTES\.md&quot;\. Text to come before this is in a file
called &quot;\_headNotes\.md&quot; and any text to come after in a file called
&quot;\_tailNotes\.md&quot;\.


## Markdown snippets
This program will discover any modules that the program being documented uses\.
Having found these packages it will find any whose name starts with one of the
standard prefixes \(by default: &apos;github\.com/nickwells/&apos;\) and if the
package&apos;s module directory contains a file called &apos;\_snippet\.md&apos;
then the contents of that file will be added to the end of the main documentary
Markdown file \(ending &apos;\.DOC\.md&apos;\)



Note that you can add to the standard prefixes by passing the
&apos;snippet\-mod\-prefix&apos; parameter\. Similarly, you can exclude specific
modules by passing the &apos;snippet\-mod\-skip&apos; parameter\.


