package pprint

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func Print(val interface{}) {
	fmt.Println(String(val))
}

func Println(val interface{}) {
	fmt.Println(String(val))
}

func String(val interface{}) string {
	return string(Bytes(val))
}

func Bytes(val interface{}) []byte {
	buf := bytes.Buffer{}

	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.Encode(val)

	return buf.Bytes()
}
