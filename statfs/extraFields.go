//go:build !(arm && linux) && !(amd64 && linux)

package main

// addAllowedFields adds the extra allowed fields
func (prog *Prog) addAllowedFields() {}

// addFieldInfo adds the extra field info
func (prog *Prog) addFieldInfo() {}
