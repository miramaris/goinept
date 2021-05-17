// +build cgo
package main

import "C"

import (
	"github.com/miramaris/goinept/pkg/goinept"
)

//export Decrypt
func Decrypt(key string, epub string, output string) {
	goinept.DecryptEpub(key, epub, output)
}

func main() {}