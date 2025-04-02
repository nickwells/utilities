package main

import (
	"path/filepath"
	"strings"
)

// populateEnv generates the environment for the generated program
func (g *gosh) populateEnv(env []string) []string {
	newEnv := []string{}

	replace := map[string]string{
		"_": filepath.Join(g.goshDir, g.execName),
	}

	for _, ev := range g.env {
		kv := strings.Split(ev, "=")
		replace[kv[0]] = kv[1]
	}

	for _, ev := range env {
		kv := strings.Split(ev, "=")
		k := kv[0]

		if v, ok := replace[k]; ok {
			ev = k + "=" + v
			delete(replace, k)
		}

		newEnv = append(newEnv, ev)
	}

	for k, v := range replace {
		newEnv = append(newEnv, k+"="+v)
	}

	return newEnv
}
