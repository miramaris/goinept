package main

import (
	"syscall/js"

	"github.com/miramaris/goinept/pkg/goinept"
)

func decryptEpub(this js.Value, args []js.Value) interface{} {
	// args [keyFileBytes, epubFileBytes, callback(err, outputBytes)]
	keyFileBytes := args[0]
	epubFileBytes := args[1]
	cb := args[2]

	keyFileBuffer := make([]byte, keyFileBytes.Get("length").Int())
	js.CopyBytesToGo(keyFileBuffer, keyFileBytes)

	epubFileBuffer := make([]byte, epubFileBytes.Get("length").Int())
	js.CopyBytesToGo(epubFileBuffer, epubFileBytes)

	outputBytes := goinept.DecryptEpubFromBytes(keyFileBuffer, epubFileBuffer)

	dst := js.Global().Get("Uint8Array").New(outputBytes.Len())
	js.CopyBytesToJS(dst, outputBytes.Bytes())
	cb.Invoke(js.Null(), dst)
	return nil
}

func main() {
	c := make(chan bool)
	js.Global().Set("decryptEpub", js.FuncOf(decryptEpub))
	<-c
}