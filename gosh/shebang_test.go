package main

import (
	"testing"
)

func TestShebangBytes(t *testing.T) {
	for in, want := range shebangTests {
		script, config := shebangStrip([]byte(in))

		if string(script) != want.script {
			t.Errorf("%q: bad script: got %q, wanted %q",
				in, script, want.script)
		}

		if string(config) != want.config {
			t.Errorf("%q: bad config: got %q, wanted %q",
				in, config, want.config)
		}
	}
}

var shebangTests = map[string]struct{ script, config string }{
	`#!`:   {},
	`#!\n`: {},
	`#!gosh -shebang
`: {},
	`#!gosh -shebang
text`: {script: "text"},
	`#!gosh -shebang
` + shebangGoshParam + ` x=y
text`: {script: `text`, config: " x=y\n"},
	`#!gosh -shebang
` + shebangGoshParam + ` x=y`: {config: ` x=y`},
	` #!`: {script: ` #!`},
}
