package main

import (
	"bufio"
	"fmt"
	"github.com/ziutek/telnet"
	"os"
)

func checkErr(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s USER PASSWD\n", os.Args[0])
	}
	user, passwd := os.Args[1], os.Args[2]

	t, err := telnet.Dial("tcp", "127.0.0.1:23")
	checkErr(err)
	t.SetTextMode(true)
	r := bufio.NewReader(t)

	line, err := r.ReadString(':')
	checkErr(err)
	fmt.Print(line)

	_, err = fmt.Fprintln(t, user)
	checkErr(err)

	line, err = r.ReadString(':')
	checkErr(err)
	fmt.Print(line)

	_, err = fmt.Fprintln(t, passwd)
	checkErr(err)

	line, err = r.ReadString('$')
	checkErr(err)
	fmt.Print(line)

	_, err = fmt.Fprintln(t, "ls -l")
	checkErr(err)

	for {
		line, err = r.ReadString('$')
		checkErr(err)
		fmt.Print(line)
	}
}
