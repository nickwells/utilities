package main

import (
	"io"
	"os"
)

// readFromStdin will return the text read from os.Stdin
func readFromStdin(_ *gosh, _ string) ([]string, error) {
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, err
	}

	return []string{string(b)}, nil
}
