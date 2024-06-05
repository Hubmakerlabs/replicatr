package text

import (
	"os"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"mleku.net/slog"
)

var log, chk = slog.New(os.Stderr)

// FirstHexCharToValue returns the hex value of a provided character from the
// first place in an 8 bit value of two characters.
//
// Two of these functions exist to minimise the computation cost, thus doubling
// the memory cost in the switch lookup table.
func FirstHexCharToValue(in byte) (out byte) {
	switch in {
	case '0':
		return 0x00
	case '1':
		return 0x10
	case '2':
		return 0x20
	case '3':
		return 0x30
	case '4':
		return 0x40
	case '5':
		return 0x50
	case '6':
		return 0x60
	case '7':
		return 0x70
	case '8':
		return 0x80
	case '9':
		return 0x90
	case 'a':
		return 0xa0
	case 'b':
		return 0xb0
	case 'c':
		return 0xc0
	case 'd':
		return 0xd0
	case 'e':
		return 0xe0
	case 'f':
		return 0xf0
	case 'A':
		return 0xA0
	case 'B':
		return 0xB0
	case 'C':
		return 0xC0
	case 'D':
		return 0xD0
	case 'E':
		return 0xE0
	case 'F':
		return 0xF0
	default:
		return 0
	}
}

// SecondHexCharToValue returns the hex value of a provided character from the
// second (last) place in an 8 bit value.
func SecondHexCharToValue(in byte) (out byte) {
	switch in {
	case '0':
		return 0x0
	case '1':
		return 0x1
	case '2':
		return 0x2
	case '3':
		return 0x3
	case '4':
		return 0x4
	case '5':
		return 0x5
	case '6':
		return 0x6
	case '7':
		return 0x7
	case '8':
		return 0x8
	case '9':
		return 0x9
	case 'a':
		return 0xa
	case 'b':
		return 0xb
	case 'c':
		return 0xc
	case 'd':
		return 0xd
	case 'e':
		return 0xe
	case 'f':
		return 0xf
	case 'A':
		return 0xA
	case 'B':
		return 0xB
	case 'C':
		return 0xC
	case 'D':
		return 0xD
	case 'E':
		return 0xE
	case 'F':
		return 0xF
	default:
		return 0
	}
}

// UnescapeByteString scans a string assumed to be UTF-8 for escaped UTF-8
// characters that must be escaped for JSON/HTML encoding. This means octal
// `\xxx` unicode backslash escapes \uXXXX and \UXXXX
func UnescapeByteString(bs []byte) (o []byte) {
	if len(bs) == 0 {
		return
	}
	// log.T.F("unescaping '%s'", bs)
	in := NewBuffer(bs)  // read side
	out := NewBuffer(bs) // write side
	var err error
	var segment []byte
	var c byte
next:
	for {
		// find the first escape character.
		// start := in.Pos
		if segment, err = in.ReadUntil('\\'); err != nil {
			// log.T.F("'%s' || '%s'", string(in.Head()), string(in.Tail()))
			if len(segment) > 0 {
				// log.T.F("'%s'", string(segment))
				if err = out.WriteBytes(segment); chk.D(err) {
					break next
				}
			}
			break next
		}
		// log.D.F("'%s'/'%s' '%s'",
		// 	string(in.Buf[start:in.Pos]),
		// 	segment,
		// 	string(in.Buf[in.Pos:]),
		// )
		if len(segment) > 0 {
			// write the segment to the out side
			if err = out.WriteBytes(segment); chk.D(err) {
				break next
			}
		}
		// skip the backslash
		in.Pos++
		// get the next byte to check for a 'u'
		if c, err = in.Read(); chk.D(err) {
			break next
		}
		// log.D.F("'%s'", string(c))
		switch c {
		case 'u':
			// we are only handling 8 bit escapes so we must see 2 0s before two
			// hex digits.
			for i := 2; i < 4; i++ {
				if c, err = in.Read(); chk.D(err) {
					break next
				}
				if c != '0' {
					// if it is not numbers after the `u`, just advance the
					// cursor.
					out.Pos += i
					in.Pos = out.Pos
					continue next
				}
			}
			// first two characters were zeroes, so now we can read the hex
			// value.
			var charByte byte
			for i := 4; i < 6; i++ {
				if c, err = in.Read(); chk.D(err) {
					break next
				}
				switch c {
				case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a',
					'b', 'c', 'd', 'e', 'f', 'A', 'B', 'C', 'D', 'E', 'F':
					// 4th char in escape is even, second is odd.
					if i%2 == 0 {
						charByte = FirstHexCharToValue(c)
					} else {
						charByte += SecondHexCharToValue(c)
					}
				default:
					// if either of these two are not hex, advance cursor and
					// continue
					log.D.Ln("skip")
					out.Pos += i
					in.Pos = out.Pos
					continue next
				}

			}
			// we now have the character to write into the out buffer.
			if err = out.Write(charByte); chk.D(err) {
				break next
			}
		default:
			// log.D.F("not u escape '%s'", string(c))
			writeChar := c
			switch c {
			case QuotationMark:
				writeChar = QuotationMark
			case 'b':
				writeChar = Backspace
			case 't':
				writeChar = Tab
			case ReverseSolidus:
				writeChar = ReverseSolidus
			case 'n':
				writeChar = LineFeed
			case 'f':
				writeChar = FormFeed
			case 'r':
				writeChar = CarriageReturn
			case ' ':
				writeChar = Space
			default:
				log.D.F("UNESCAPE \\%s", string(c))
			}
			// we now have the character to write into the out buffer.
			if err = out.Write(writeChar); chk.D(err) {
				break next
			}

			// log.D.F("UNESCAPE '%s' '%s' '%s' -> '%s' '%s'", string(bs),
			// 	string(in.Head()), string(in.Tail()),
			// 	string(out.Head()), string(out.Tail()))
		}
	}
	// when we get to here, the cursor marks the end of the unescaped string.
	o = out.Head()
	// truncate the original as well so it can't be mistakenly re-used
	bs = o
	return
}

// unquoteBytes is taken directly from encoding/json as it is unfortunately not
// exposed for independent use.
//
// currently unused and probably
func unquoteBytes(s []byte) (t []byte, ok bool) {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return
	}
	s = s[1 : len(s)-1]

	// Check for unusual characters. If there are none,
	// then no unquoting is needed, so return a slice of the
	// original bytes.
	r := 0
	for r < len(s) {
		c := s[r]
		if c == '\\' || c == '"' || c < ' ' {
			break
		}
		if c < utf8.RuneSelf {
			r++
			continue
		}
		rr, size := utf8.DecodeRune(s[r:])
		if rr == utf8.RuneError && size == 1 {
			break
		}
		r += size
	}
	if r == len(s) {
		return s, true
	}

	b := make([]byte, len(s)+2*utf8.UTFMax)
	w := copy(b, s[0:r])
	for r < len(s) {
		// Out of room? Can only happen if s is full of
		// malformed UTF-8 and we're replacing each
		// byte with RuneError.
		if w >= len(b)-2*utf8.UTFMax {
			nb := make([]byte, (len(b)+utf8.UTFMax)*2)
			copy(nb, b[0:w])
			b = nb
		}
		switch c := s[r]; {
		case c == '\\':
			r++
			if r >= len(s) {
				return
			}
			switch s[r] {
			default:
				return
			case '"', '\\', '/', '\'':
				b[w] = s[r]
				r++
				w++
			case 'b':
				b[w] = '\b'
				r++
				w++
			case 'f':
				b[w] = '\f'
				r++
				w++
			case 'n':
				b[w] = '\n'
				r++
				w++
			case 'r':
				b[w] = '\r'
				r++
				w++
			case 't':
				b[w] = '\t'
				r++
				w++
			case 'u':
				r--
				rr := getu4(s[r:])
				if rr < 0 {
					return
				}
				r += 6
				if utf16.IsSurrogate(rr) {
					rr1 := getu4(s[r:])
					if dec := utf16.DecodeRune(rr,
						rr1); dec != unicode.ReplacementChar {
						// A valid pair; consume.
						r += 6
						w += utf8.EncodeRune(b[w:], dec)
						break
					}
					// Invalid surrogate; fall back to replacement rune.
					rr = unicode.ReplacementChar
				}
				w += utf8.EncodeRune(b[w:], rr)
			}

		// Quote, control characters are invalid.
		case c == '"', c < ' ':
			return

		// ASCII
		case c < utf8.RuneSelf:
			b[w] = c
			r++
			w++

		// Coerce to well-formed UTF-8.
		default:
			rr, size := utf8.DecodeRune(s[r:])
			r += size
			w += utf8.EncodeRune(b[w:], rr)
		}
	}
	return b[0:w], true
}

// getu4 decodes \uXXXX from the beginning of s, returning the hex value,
// or it returns -1.
func getu4(s []byte) rune {
	if len(s) < 6 || s[0] != '\\' || s[1] != 'u' {
		return -1
	}
	var r rune
	for _, c := range s[2:6] {
		switch {
		case '0' <= c && c <= '9':
			c = c - '0'
		case 'a' <= c && c <= 'f':
			c = c - 'a' + 10
		case 'A' <= c && c <= 'F':
			c = c - 'A' + 10
		default:
			return -1
		}
		r = r*16 + rune(c)
	}
	return r
}
