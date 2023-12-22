package text

import (
	log2 "mleku.online/git/log"
)

var (
	log   = log2.GetLogger()
	fails = log.D.Chk
)

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
	in := New(bs)  // read side
	out := New(bs) // write side
	var e error
	var segment []byte
	var c byte
next:
	for {
		// find the first escape character.
		// start := in.Pos
		if segment, e = in.ReadUntil('\\'); e != nil {
			// log.D.F("'%s'", string(in.Buf[start:]))
			if len(segment) > 0 {
				if e = out.WriteBytes(segment); fails(e) {
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
			if e = out.WriteBytes(segment); fails(e) {
				break next
			}
		}
		// skip the backslash
		in.Pos++
		// get the next byte to check for a 'u'
		if c, e = in.Read(); fails(e) {
			break next
		}
		// log.D.F("'%s'", string(c))
		switch c {
		case 'u':
			// we are only handling 8 bit escapes so we must see 2 0s before two
			// hex digits.
			for i := 2; i < 4; i++ {
				if c, e = in.Read(); fails(e) {
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
				if c, e = in.Read(); fails(e) {
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
			if e = out.Write(charByte); fails(e) {
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
			if e = out.Write(writeChar); fails(e) {
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
