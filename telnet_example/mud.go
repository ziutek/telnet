/*-----------------------------------------------------------------
  Telnet Server example for golang telnet:

  https://github.com/ziutek/telnet

  (altered/modified/extended from original client example)
  by mahan on 20130606  
-----------------------------------------------------------------*/
package main

import (
  "fmt"
  "github.com/ziutek/telnet"
  "log"
  "time"
  "net"
  "strings"
)

const timeout = 10 * time.Second

func checkErr(err error) {
  if err != nil {
    log.Fatalln("Error:", err)
  }
}

//This is where you handle the flow of incoming connections.
func handleConnection(c net.Conn) {
  conn, _ := telnet.NewConn(c)
  checkErr(c.SetDeadline(time.Now().Add(timeout)))

  fmt.Println("------------------------ Handle Connection --------------------------")
  sendln(conn, ":D (LolMUd 6.53) \x1b[31;1mDangereous Dragons\x1b[0m and stuff be here.\n")
  send(conn, "What is your name, traveller? ")
  username, err := input(conn)
  checkErr(err)
  sendln(conn, "")
  sendln(conn, fmt.Sprintf("Hello there, %s. Sorry but I'll have to disconnect you until the game is ready.", username))
  fmt.Println("User connected: ", username)
  conn.Close()
  fmt.Println("Connection for", username, "disconnected.")
}

//Get a line of text from the user
func input(t *telnet.Conn) (string, error) {
  username, err := t.ReadString('\r')
  username = strings.Trim(username, "\r\f")
  return username, err
}

//Send a row of text to the client (without ending the line)
func send(t *telnet.Conn, s string) {
  buf := make([]byte, len(s))
  copy(buf, s)
  _, err := t.Write(buf)
  checkErr(err)
}

//Send a row of text to the client + EOL
func sendln(t *telnet.Conn, s string) {
  s += "\n"
  send(t, s)
}

//Generic TCP-Server startup
func main() {
  fmt.Println("Mud started. Write 'telnet 127.0.0.1 4000' in another terminal to connect.")
  ln, err := net.Listen("tcp", ":4000")
  checkErr(err)
  for {
    conn, err := ln.Accept()
    checkErr(err)
    go handleConnection(conn)
  }
}
