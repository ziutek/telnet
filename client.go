package telnet

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net"
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

// Client implements net.Conn interface for Telnet protocol plus some set Telnet
// specific methods
type Client struct {
	net.Conn
	r *bufio.Reader

	textMode bool

	cliSuppressGoAhead bool
	cliEcho            bool

	UnixWriteMode bool
}

func NewClient(conn net.Conn) (*Client, error) {
	c := Client{
		Conn: conn,
		r:    bufio.NewReaderSize(conn, 256),
	}
	return &c, nil
}

func Dial(network, addr string) (*Client, error) {
	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	return NewClient(conn)
}

func (c *Client) do(option byte) error {
	//log.Println("do:", option)
	_, err := c.Conn.Write([]byte{cmdIAC, cmdDo, option})
	return err
}

func (c *Client) dont(option byte) error {
	//log.Println("dont:", option)
	_, err := c.Conn.Write([]byte{cmdIAC, cmdDont, option})
	return err
}

func (c *Client) will(option byte) error {
	//log.Println("will:", option)
	_, err := c.Conn.Write([]byte{cmdIAC, cmdWill, option})
	return err
}

func (c *Client) wont(option byte) error {
	//log.Println("wont:", option)
	_, err := c.Conn.Write([]byte{cmdIAC, cmdWont, option})
	return err
}

func (c *Client) cmd(cmd byte) error {
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
		// If echo need to be disabled at server side client
		// need to signal server that it will use local echo
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
				err = c.will(o)
			}
		case cmdDont:
			if c.cliSuppressGoAhead {
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

func (c *Client) ReadByte() (byte, error) {
loop:
	b, err := c.r.ReadByte()
	if err != nil {
		return 0, err
	}
	if b == cmdIAC {
		b, err = c.r.ReadByte()
		if err != nil {
			return 0, err
		}
		if b != cmdIAC {
			err = c.cmd(b)
			if err != nil {
				return 0, err
			}
			goto loop
		}
	}
	return b, nil
}

func (c *Client) Read(buf []byte) (int, error) {
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

func (c *Client) Write(buf []byte) (int, error) {
	search := "\xff"
	if c.UnixWriteMode {
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
		log.Println("###idx", i)
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
