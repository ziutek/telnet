package main

import (
	"log"
	"os"

	"github.com/ziutek/telnet"
)

func checkErr(err error) {
	if err != nil {
		log.Fatalln("Error:", err)
	}
}

func main() {
	t, err := telnet.Dial("tcp", "bat.org:23")
	checkErr(err)

	buf := make([]byte, 512)
	for {
		n, err := t.Read(buf) // Use raw read to find issue #15.
		os.Stdout.Write(buf[:n])
		checkErr(err)
	}
}
