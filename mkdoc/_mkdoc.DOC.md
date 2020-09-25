<!-- Created by mkdoc DO NOT EDIT. -->

# mkdoc

This creates markdown documentation for any Go program which uses the param
package \(github\.com/nickwells/param\.mod/\*/param\)\. It will generate a
markdown file containing examples if the program has examples and it will
generate a file containing references if the program has references\. It will
generate a main doc file which will have links to the examples and references
files if they exist\. This main doc file should then be linked to from the
README\.md file\.

You can give additional text to be printed at the end of each of the markdown
files in the following files \(none of which need to exist\):
&apos;\_tailDoc\.md&apos;, &apos;\_tailExamples\.md&apos;,
&apos;\_tailReferences\.md&apos;



