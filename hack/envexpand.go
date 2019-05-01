// +build ignore

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

func main() {
	log.SetFlags(0)
	b, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(os.ExpandEnv(string(b)))
}
