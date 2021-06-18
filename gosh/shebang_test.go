package main

import (
	"bytes"
	"regexp"
	"testing"
)

func BenchmarkShebangRx(b *testing.B) {
	re := regexp.MustCompile("^#![^\n]*(\n|$)")
	inWants := mkBytes(shebangTests)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, inWant := range inWants {
			got := re.ReplaceAll(inWant[0], nil)
			if !bytes.Equal(got, inWant[1]) {
				b.Fatalf("%q: got %q, wanted %q",
					string(inWant[0]), string(got), string(inWant[1]))
			}
		}
	}
}

func BenchmarkShebangRxEach(b *testing.B) {
	inWants := mkBytes(shebangTests)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, inWant := range inWants {
			re := regexp.MustCompile("^#![^\n]*(\n|$)")
			got := re.ReplaceAll(inWant[0], nil)
			if !bytes.Equal(got, inWant[1]) {
				b.Fatalf("%q: got %q, wanted %q",
					string(inWant[0]), string(got), string(inWant[1]))
			}
		}
	}
}

func BenchmarkShebangBytes(b *testing.B) {
	inWants := mkBytes(shebangTests)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, inWant := range inWants {
			got := shebangRemove(inWant[0])
			if !bytes.Equal(got, inWant[1]) {
				b.Fatalf("%q: got %q, wanted %q",
					string(inWant[0]), string(got), string(inWant[1]))
			}
		}
	}
}

func mkBytes(m map[string]string) [][2][]byte {
	arr := make([][2][]byte, 0, len(m))
	for k, v := range m {
		arr = append(arr, [2][]byte{[]byte(k), []byte(v)})
	}
	return arr
}

func TestShebangBytes(t *testing.T) {
	for in, want := range shebangTests {
		got := string(shebangRemove([]byte(in)))
		if got != want {
			t.Errorf("%q: got %q, wanted %q", in, got, want)
		}
	}
}

var shebangTests = map[string]string{
	"#!":             "",
	"#!\n":           "",
	"#asd!\r\n":      "#asd!\r\n",
	"#!/asdsad\nyes": "yes",
	" #!\n":          " #!\n",
}
