// Package telnet provides simple interface for interacting with Telnet
// connection.
package telnet

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"unicode"
)

const (
	CR = byte('\r')
	LF = byte('\n')
)

const (
	cmdSE   = 240
	cmdNOP  = 241
	cmdData = 242

	cmdBreak = 243
	cmdGA    = 249
	cmdSB    = 250

	cmdWill = 251
	cmdWont = 252
	cmdDo   = 253
	cmdDont = 254

	cmdIAC = 255
)

const (
	optEcho            = 1
	optSuppressGoAhead = 3
)

// Conn implements net.Conn interface for Telnet protocol plus some set of
// Telnet specific methods.
type Conn struct {
	net.Conn
	r *bufio.Reader

	unixWriteMode bool

	cliSuppressGoAhead bool
	cliEcho            bool
}

func NewConn(conn net.Conn) (*Conn, error) {
	c := Conn{
		Conn: conn,
		r:    bufio.NewReaderSize(conn, 256),
	}
	return &c, nil
}

func Dial(network, addr string) (*Conn, error) {
	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	return NewConn(conn)
}

// SetUnixWriteMode sets flag that applies only to the Write method.
// If set, Write converts any '\n' (LF) to '\r\n' (CR LF).
func (c *Conn) SetUnixWriteMode(uwm bool) {
	c.unixWriteMode = uwm
}

func (c *Conn) do(option byte) error {
	//log.Println("do:", option)
	_, err := c.Conn.Write([]byte{cmdIAC, cmdDo, option})
	return err
}

func (c *Conn) dont(option byte) error {
	//log.Println("dont:", option)
	_, err := c.Conn.Write([]byte{cmdIAC, cmdDont, option})
	return err
}

func (c *Conn) will(option byte) error {
	//log.Println("will:", option)
	_, err := c.Conn.Write([]byte{cmdIAC, cmdWill, option})
	return err
}

func (c *Conn) wont(option byte) error {
	//log.Println("wont:", option)
	_, err := c.Conn.Write([]byte{cmdIAC, cmdWont, option})
	return err
}

func (c *Conn) cmd(cmd byte) error {
	switch cmd {
	case cmdGA:
		return nil
	case cmdDo, cmdDont, cmdWill, cmdWont:
	default:
		return fmt.Errorf("unknwn command: %d", cmd)
	}
	// Read an option
	o, err := c.r.ReadByte()
	if err != nil {
		return err
	}
	//log.Println("received cmd:", cmd, o)
	switch o {
	case optEcho:
		// Accept any echo configuration.
		// TODO: enable/disable echo on server side.
		switch cmd {
		case cmdDo:
			if !c.cliEcho {
				c.cliEcho = true
				err = c.will(o)
			}
		case cmdDont:
			if c.cliEcho {
				c.cliEcho = false
				err = c.wont(o)
			}
		case cmdWill:
			err = c.do(o)
		case cmdWont:
			err = c.dont(o)
		}
	case optSuppressGoAhead:
		// We don't use GA so can allways accept every configuration
		switch cmd {
		case cmdDo:
			if !c.cliSuppressGoAhead {
				c.cliSuppressGoAhead = true
				err = c.will(o)
			}
		case cmdDont:
			if c.cliSuppressGoAhead {
				c.cliSuppressGoAhead = false
				err = c.wont(o)
			}
		case cmdWill:
			err = c.do(o)
		case cmdWont:
			err = c.dont(o)

		}
	default:
		// Deny any other option
		switch cmd {
		case cmdDo:
			err = c.wont(o)
		case cmdDont:
			// nop
		case cmdWill, cmdWont:
			err = c.dont(o)
		}
	}
	return err
}

func (c *Conn) tryReadByte() (b byte, retry bool, err error) {
	b, err = c.r.ReadByte()
	if err != nil || b != cmdIAC {
		return
	}
	b, err = c.r.ReadByte()
	if err != nil {
		return
	}
	if b != cmdIAC {
		err = c.cmd(b)
		if err != nil {
			return
		}
		retry = true
	}
	return
}

// ReadByte works like bufio.ReadByte
func (c *Conn) ReadByte() (b byte, err error) {
	retry := true
	for retry && err == nil {
		b, retry, err = c.tryReadByte()
	}
	return
}

// ReadRune works like bufio.ReadRune
func (c *Conn) ReadRune() (r rune, size int, err error) {
loop:
	r, size, err = c.r.ReadRune()
	if err != nil {
		return
	}
	if r != unicode.ReplacementChar || size != 1 {
		// Properly readed rune
		return
	}
	// Bad rune
	err = c.r.UnreadRune()
	if err != nil {
		return
	}
	// Read telnet command or escaped IAC
	_, retry, err := c.tryReadByte()
	if err != nil {
		return
	}
	if retry {
		// This bad rune was a begining of telnet command. Try read next rune.
		goto loop
	}
	// Return escaped IAC as unicode.ReplacementChar
	return
}

// Read implements io.Reader.Read
func (c *Conn) Read(buf []byte) (int, error) {
	var n int
	for n < len(buf) {
		b, err := c.ReadByte()
		if err != nil {
			return n, err
		}
		//log.Printf("char: %d %q", b, b)
		buf[n] = b
		n++
		if c.r.Buffered() == 0 {
			// Try don't block if can return some data
			break
		}
	}
	return n, nil
}

// ReadBytes works like bufio.ReadBytes
func (c *Conn) ReadBytes(delim byte) ([]byte, error) {
	var line []byte
	for {
		b, err := c.ReadByte()
		if err != nil {
			return nil, err
		}
		line = append(line, b)
		if b == delim {
			break
		}
	}
	return line, nil
}

// SkipBytes works like ReadBytes but skips all read data.
func (c *Conn) SkipBytes(delim byte) error {
	for {
		b, err := c.ReadByte()
		if err != nil {
			return err
		}
		if b == delim {
			break
		}
	}
	return nil
}

// ReadString works like bufio.ReadString
func (c *Conn) ReadString(delim byte) (string, error) {
	bytes, err := c.ReadBytes(delim)
	return string(bytes), err
}

func (c *Conn) readUntil(read bool, patterns ...string) ([]byte, int, error) {
	if len(patterns) == 0 {
		return nil, 0, nil
	}
	p := make([]string, len(patterns))
	for i, s := range patterns {
		if len(s) == 0 {
			return nil, 0, nil
		}
		p[i] = s
	}
	var line []byte
	for {
		b, err := c.ReadByte()
		if err != nil {
			return nil, 0, err
		}
		if read {
			line = append(line, b)
		}
		for i, s := range p {
			if s[0] == b {
				if len(s) == 1 {
					return line, i, nil
				}
				p[i] = s[1:]
			} else {
				p[i] = patterns[i]
			}
		}
	}
	panic(nil)
}

// ReadUntilIndex reads from connection until one of patterns occurs. Returns
// read data and an index of pattern or error.
func (c *Conn) ReadUntilIndex(patterns ...string) ([]byte, int, error) {
	return c.readUntil(true, patterns...)
}

// ReadUntil works like ReadUntilIndex but don't return pattern index.
func (c *Conn) ReadUntil(patterns ...string) ([]byte, error) {
	d, _, err := c.readUntil(true, patterns...)
	return d, err
}

// SkipUntilIndex works like ReadUntilIndex but skips all read data.
func (c *Conn) SkipUntilIndex(patterns ...string) (int, error) {
	_, i, err := c.readUntil(false, patterns...)
	return i, err
}

// SkipUntil works like ReadUntil but skips all read data.
func (c *Conn) SkipUntil(patterns ...string) error {
	_, _, err := c.readUntil(false, patterns...)
	return err
}

// Write is for implement io.Writer
func (c *Conn) Write(buf []byte) (int, error) {
	search := "\xff"
	if c.unixWriteMode {
		search = "\xff\n"
	}
	var (
		n   int
		err error
	)
	for len(buf) > 0 {
		var k int
		i := bytes.IndexAny(buf, search)
		if i == -1 {
			k, err = c.Conn.Write(buf)
			n += k
			break
		}
		k, err = c.Conn.Write(buf[:i])
		n += k
		if err != nil {
			break
		}
		switch buf[i] {
		case LF:
			k, err = c.Conn.Write([]byte{CR, LF})
		case cmdIAC:
			k, err = c.Conn.Write([]byte{cmdIAC, cmdIAC})
		}
		n += k
		if err != nil {
			break
		}
		buf = buf[i+1:]
	}
	return n, err
}
