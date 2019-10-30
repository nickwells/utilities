/*

The fileChecker command finds all files in a given directory with a given
extension and compares them against corresponding files without the
extension. Then the user is prompted to delete the file with the extension.

It is most useful in conjunction with the testhelper package. The testhelper
package will retain the original contents of a golden file in a file of the
same name with an extension of '.orig'. This command will help you to review
the changes and tidy up afterwards.

*/
package main
