// +build ignore

package main

import (
	"fmt"
	"log"

	"bazil.org/bazil/util/trylisten"
)

func main() {
	l, err := trylisten.Listen("tcp", "localhost:1234")
	if err != nil {
		fmt.Printf("%T %#v", err, err)
		log.Fatal(err)
	}
	fmt.Println(l.Addr())
}
