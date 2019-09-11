package util

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/kylelemons/godebug/diff"
)

func WriteTempFile(data []byte) (string, error) {
	tmpfile, err := ioutil.TempFile("", "example")
	if err != nil {
		return "", fmt.Errorf("could create temp file: %v", err)
	}

	if _, err := tmpfile.Write(data); err != nil {
		return "", fmt.Errorf("could write temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		return "", fmt.Errorf("could close temp file: %v", err)
	}
	return tmpfile.Name(), nil
}

// Render renders the slice of chunks into a representation that prefixes
// the lines with '+', '-', or ' ' depending on whether the line was added,
// removed, or equal (respectively).
func Render(chunks []diff.Chunk) string {
	buf := new(strings.Builder)
	for _, c := range chunks {
		for _, line := range c.Added {
			fmt.Fprintf(buf, "+%s\n", line)
		}
		for _, line := range c.Deleted {
			fmt.Fprintf(buf, "-%s\n", line)
		}
		if len(c.Equal) < 6 {
			for _, line := range c.Equal {
				fmt.Fprintf(buf, " %s\n", line)
			}
		} else {
			fmt.Fprintf(buf, " %s\n", c.Equal[0])
			fmt.Fprintf(buf, " %s\n", c.Equal[1])
			fmt.Fprintf(buf, "...\n")
			fmt.Fprintf(buf, " %s\n", c.Equal[len(c.Equal)-2])
			fmt.Fprintf(buf, " %s\n", c.Equal[len(c.Equal)-1])
		}
	}
	return strings.TrimRight(buf.String(), "\n")
}
