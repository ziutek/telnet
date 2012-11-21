package main

import (
	"fmt"
	"github.com/ziutek/telnet"
	"log"
	"os"
	"time"
)

func checkErr(err error) {
	if err != nil {
		log.Fatalln("Error:", err)
	}
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s USER PASSWD\n", os.Args[0])
		return
	}
	user, passwd := os.Args[1], os.Args[2]

	t, err := telnet.Dial("tcp", "127.0.0.1:23")
	checkErr(err)
	err = t.SetDeadline(time.Now().Add(5 * time.Second))
	checkErr(err)
	t.SetUnixWriteMode(true)

	checkErr(t.SkipUntil("login: "))

	_, err = fmt.Fprintln(t, user)
	checkErr(err)

	checkErr(t.SkipUntil("ssword: "))

	_, err = fmt.Fprintln(t, passwd)
	checkErr(err)

	checkErr(t.SkipUntil("$ "))

	_, err = fmt.Fprintln(t, "ls -l")
	checkErr(err)

	ls, err := t.ReadUntil("$ ")
	checkErr(err)
	os.Stdout.Write(ls)
}
