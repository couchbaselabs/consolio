package main

import (
	"bufio"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/dustin/gojson"
)

type state int

const (
	inside state = iota
	outside
	escaped
)

var space = make([]byte, utf8.UTFMax)

type transformer func(string) string

type machine struct {
	state state
	trans transformer
	buf   []byte
	out   *bufio.Writer
}

func identity(s string) string {
	return s
}

func (m *machine) tick(inchar rune) error {
	incomingState := m.state

	if m.state == escaped {
		m.state = inside
	} else {
		switch inchar {
		case '"':
			switch m.state {
			case outside:
				m.state = inside
			case inside:
				// Do stuff with content of Stringbuf
				m.state = outside
			}
		case '\\':
			if m.state == inside {
				m.state = escaped
			}
		}
	}

	if (m.state == inside || m.state == escaped) ||
		(incomingState == inside || incomingState == escaped) {
		l := utf8.RuneLen(inchar)
		m.buf = append(m.buf, space[:l]...)
		gl := utf8.EncodeRune(m.buf[len(m.buf)-l:], inchar)
		if l != gl {
			panic("length mismatch")
		}
	}
	if m.state == outside && incomingState == outside {
		m.out.WriteRune(inchar)
	}

	if m.state == outside && incomingState == inside {
		s, ok := json.UnquoteBytes(m.buf)
		if !ok {
			return fmt.Errorf("Error parsing %s as json string",
				m.buf)
		}
		o, err := json.Marshal(m.trans(string(s)))
		if err != nil {
			return err
		}
		_, err = m.out.Write(o)
		if err != nil {
			return err
		}
		m.buf = m.buf[:0]
	}

	return nil

}

func newMachine(out io.Writer, trans transformer) *machine {
	return &machine{outside, trans, nil, bufio.NewWriter(out)}
}

func (m *machine) rage(in io.RuneReader) error {
	runes := 0
	for {
		r, _, err := in.ReadRune()
		switch err {
		case nil:
			runes++
			err = m.tick(r)
			if err != nil {
				return err
			}
		case io.EOF:
			return m.out.Flush()
		default:
			return err
		}
	}
}

func rewriteJson(in io.Reader, trans transformer) io.Reader {
	pr, pw := io.Pipe()

	br, ok := in.(io.RuneReader)
	if !ok {
		br = bufio.NewReader(in)
	}

	go func() {
		m := newMachine(pw, trans)
		pw.CloseWithError(m.rage(br))
	}()

	return pr
}
