// +build !cgo
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/miramaris/goinept/pkg/goinept"
)

func main() {
	input := flag.String("input", "", "path to encrypted epub file")
	key := flag.String("key", "", "path to .der decryption key")
	output := flag.String("output", "", "path to desired output file")
	level := flag.Int("level", -1, "desired compression level [0-9]")

	flag.Parse()

	if (*input == "") {
		fmt.Println("-input not specified, must be a path to an encrypted epub file")
		os.Exit(1)
	}

	if (*key == "") {
		fmt.Println("-key not specified, must be a path to a .der decryption key")
		os.Exit(1)
	}

	if (*output == "") {
		fmt.Println("-output not specified, must be a path to the desired output file")
		os.Exit(1)
	}

	if (*level < -1 || *level > 9) {
		fmt.Println("-level must be a number between 1 and 9 inclusive")
	}

	fmt.Printf("Decrypting %s using key at %s.\n", *input, *key)
	goinept.DecryptEpub(*key, *input, *output, *level)

	fmt.Printf("Wrote decrypted EPUB to %s.\n", *output)
}