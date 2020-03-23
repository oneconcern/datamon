// +build ignore

package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	if len(os.Args) == 0 {
		return
	}
	input := strings.TrimSpace(os.Args[1])
	t, err := time.Parse(time.RFC3339, input)
	if err != nil {
		t, err = time.Parse(time.RFC3339Nano, input)
		if err != nil {
			t, err = time.Parse("2006-01-02 15:04:05.9999 +0000 UTC", input)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		}
	}
	fmt.Printf("%d\n", t.UnixNano())
}
