package text

import "unicode/utf8"

// The character constants are used as their names. IDEs with inlays expanding
// the values will demonstrate the equivalence of these with the same decimal
// UTF-8 value, thus the secondary items with their Go rune equivalents.
//
// The human readable forms are given in order to educate more than anything
// else. The same symbols can be used in regular Go double quoted "" strings to
// indicate the same character.
//
// Different rules apply to backtick quoted strings, which allow any character
// to be placed in a string, escaped sequences are literally interpreted instead
// of parsed to their respective bytes, and generally editors won't allow the
// placement of control characters in these strings; their purpose is allowing
// properly flowed, line-break containing strings such as embedding literal
// text. Backtick strings can contain printf formatting same as double quote
// strings.
const (
	QuotationMark    = 0x22
	QuotationMarkGo  = '"'
	ReverseSolidus   = 0x5c
	ReverseSolidusGo = '\\'
	Solidus          = 0x2f
	SolidusGo        = '/'
	Backspace        = 0x08
	BackspaceGo      = '\b'
	FormFeed         = 0x0c
	FormFeedGo       = '\f'
	LineFeed         = 0x0a
	LineFeedGo       = '\n'
	CarriageReturn   = 0x0d
	CarriageReturnGo = '\r'
	Tab              = 0x09
	TabGo            = '\t'
	Space            = 0x20
	SpaceGo          = ' '
)

// EscapeJSONStringAndWrapOld takes an arbitrary string and escapes all control
// characters as per rfc8259 section 7 https://www.rfc-editor.org/rfc/rfc8259
// (retrieved 2023-11-21):
//
//	The representation of strings is similar to conventions used in the C family
//	of programming languages. A string begins and ends with quotation marks. All
//	Unicode characters may be placed within the quotation marks, except for the
//	characters that MUST be escaped: quotation mark, reverse solidus, and the
//	control characters (U+0000 through U+001F).
//
// The string is assumed to be UTF-8 and only the above escapes are processed.
// The string will be wrapped in double quotes `"` as it is assumed that the
// string will be added to a JSON document in a place where a string is valid.
//
// The processing proceeds in two passes, first calculating the required
// expansion for the characters in the provided string, and then copying over
// and adding the required escape code expansions as indicated, to ensure that
// for very long strings only one allocation, of precisely the correct amount,
// is made.
//
// Note the iteration through the string must proceed as though the string is
// []byte rather than be interpreted using a `for _, c := range s` which will
// prompt Go to interpret the string as UTF-8 and potentially return a different
// result, this occurs on the series of characters 0-255 at a certain point due
// to UTF-8 encoding rules.
//
// One last thing to note. The stdlib function `json.Marshal` automatically runs a HTML escape processing which turns some valid characters, namely:
//
//	String values encode as JSON strings coerced to valid UTF-8, replacing
//	invalid bytes with the Unicode replacement rune. So that the JSON will be
//	safe to embed inside HTML <script> tags, the string is encoded using
//	HTMLEscape, which replaces "<", ">", "&", U+2028, and U+2029 are escaped to
//	"\u003c","\u003e", "\u0026", "\u2028", and "\u2029". This replacement can be
//	disabled when using an Encoder, by calling SetEscapeHTML(false).
//
// And so the assumption this code here makes is that backslashes need to be
// escaped, needs to have special handling to not escape the escaped, in order
// to allow custom JSON marshalers to not keep adding backslashes to valid UTF-8
// entities.
func EscapeJSONStringAndWrapOld(s string) (escaped []byte) {
	log.D.F("escaping %d %x\n'%s'", len(s), []byte(s), s)
	// first calculate the extra bytes required for the given string
	length := len(s) + 2
	for _, c := range s {
		switch {
		// handle the two character escapes `\x`
		case c == QuotationMark,
			c == ReverseSolidus,
			c == Backspace,
			c == Tab,
			c == LineFeed,
			c == FormFeed,
			c == CarriageReturn:
			length++
		// except those above 128 (see todo about UTF-8 escaping)
		case c > 128:
			length += 5
			// handle the 6 character escapes \uXXXX remaining.
		case c < Space:
			length += 5
			// remaining 8 bit values above 0x20 (Space) and below 128 are not
			// expanded
		case c >= Space:
		}
	}
	// log.D.F("preallocating %d for string of %d", length-2, len(s))
	// allocate the required bytes, and then copy in the things.
	//
	// set size to zero to allow slice runtime to handle counting the appended
	// bytes (and avoid manually tracking a cursor - saves bytes anyway, since
	// slice header already exists).
	escaped = make([]byte, 0, length)
	// add the beginning double quote character:
	escaped = append(escaped, QuotationMark)
	// Note that if this range statement uses comma syntax the value extracted
	// at each character is parsed as Unicode which will cause this conversion
	// to be incorrect according to RFC8259.
	// log.D.F("'%s'", s)
	for i := range s {
		c := s[i]
		log.D.F("%03d >%s< %d", i, string(c), c)
		switch {
		case c == QuotationMark:
			escaped = append(escaped, []byte{ReverseSolidus, QuotationMark}...)
		case c == Backspace:
			escaped = append(escaped,
				[]byte{ReverseSolidus, 'b'}...)
		case c < Tab:
			escaped = append(escaped,
				[]byte{ReverseSolidus, 'u', '0', '0', '0', '0' + byte(c)}...)
		case c == Tab:
			escaped = append(escaped,
				[]byte{ReverseSolidus, 't'}...)
		case c == LineFeed:
			escaped = append(escaped,
				[]byte{ReverseSolidus, 'n'}...)
		case c == FormFeed:
			escaped = append(escaped,
				[]byte{ReverseSolidus, 'f'}...)
		case c == CarriageReturn:
			escaped = append(escaped,
				[]byte{ReverseSolidus, 'r'}...)
		case c == ReverseSolidus:
			var notEscaped bool
			if i+1 < len(s) {
				// log.D.Ln(i, len(s))
				// look ahead to see if this is a escape:
				// log.D.F("\\ '%s'", string(s[i+1]))
			escapeCheck:
				switch s[i+1] {
				case Space:
					escaped = append(escaped,
						// []byte{ReverseSolidus, ReverseSolidus}...)
						ReverseSolidus)
				case 'u', 'U':
					// Let's just be extra careful and make sure the 2 next
					// chars are hex. If more are hex it doesn't matter. Really,
					// just the u/U is all that matters, but since it's not
					// valid escape code probably better to escape the `\` since
					// it's not properly speaking a unicode escape.
					if i < len(s)-5 {
						for x := 2; x < 4; x++ {
							switch s[i+x] {
							case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f', 'A', 'B', 'C', 'D', 'E', 'F':
							default:
								// if the first or second after the u/U are not hex,
								// it's not a unicode escape and we will double the `\`.
								notEscaped = true
							}
							if notEscaped {
								break escapeCheck
							}
						}
					}
				case '0', '1', '2', '3', '4', '5', '6', '7':
				default:
					// log.D.Ln("not an escape")
					escaped = append(escaped,
						// ReverseSolidus)
						[]byte{ReverseSolidus, ReverseSolidus}...)
					notEscaped = true
				}
				if !notEscaped {
					escaped = append(escaped,
						ReverseSolidus)
					// []byte{ReverseSolidus, ReverseSolidus}...)
				}
			}
		case c < 0x10:
			escaped = append(escaped,
				[]byte{ReverseSolidus, 'u', '0', '0', '0', 0x57 + byte(c)}...)
		case c < 0x1a:
			escaped = append(escaped,
				[]byte{ReverseSolidus, 'u', '0', '0', '1', Space + byte(c)}...)
		case c < Space:
			// all control characters not already handled in previous lines will
			// be escaped with the 4 byte escape code here.
			escaped = append(escaped,
				[]byte{ReverseSolidus, 'u', '0', '0', '1', 0x47 + byte(c)}...)
		case c >= Space:
			escaped = append(escaped, c)
		default:
			// todo: this code only deals with 8 bit ASCII, but that probably is
			//  ok? Clients will read the unescaped output as UTF8 which will
			//  preserve the raw bytes. Full UTF-8 escaping would be a lot more
			//  expensive to do.
			escaped = append(escaped, c)
			// escaped = append(escaped, []byte(fmt.Sprintf("\\u00%2x", c))...)
			// []byte{ReverseSolidus, 'u', '0', '0', '1', 0x47 + byte(c)}...)
		}
		// if c == ReverseSolidus {
		// 	log.D.F("\n1'%s' >%s< \n2'%s' -> '%s'", s[:i], string(c),
		// 		s[i+1:], string(escaped[1:]))
		// }
	}
	// add the final double quote character
	escaped = append(escaped, QuotationMark)
	return
}

// Unwrap is a dumb function that just slices off the first and last byte,
// which from the EscapeJSONStringAndWrap function is the quotes around it.
//
// This can be unsafe to run as it assumes there is at least two bytes.
//
// TODO: rewrite this all to work from []byte and optional quote wrapping.
func Unwrap(wrapped []byte) (unwrapped []byte) {
	unwrapped = wrapped[1 : len(wrapped)-1]
	return
}
func EscapeJSONStringAndWrap(s string) (escaped []byte) {
	length := len(s) + 2
	for _, c := range s {
		switch {
		// handle the two character escapes `\x`
		case c == QuotationMark,
			c == ReverseSolidus,
			c == Backspace,
			c == Tab,
			c == LineFeed,
			c == FormFeed,
			c == CarriageReturn:
			length++

			// to match what is done by escapeString, all the higher bit values
			// than 127 0x80 are left as is
			// // except those above 128 (see todo about UTF-8 escaping)
			// case c > 128:
			// 	length += 5
			// handle the 6 character escapes \uXXXX remaining.
		case c < Space:
			length += 5
			// remaining 8 bit values above 0x20 (Space) and below 128 are not
			// expanded
		case c >= Space:
		}
	}
	escaped = make([]byte, 0, length)
	// log.D.Ln(length)
	return appendString(escaped, s, false)
	// log.D.Ln(escaped)
	// return
}

// appendString is the JSON string escaping code from encoding/json but is not
// exported so it is copied here (it should remain valid enough to not need
// updating frequently - its commit history is pretty sparse and most of the
// recent changes have been to account for mostly insecure things in stupid
// javascript libraries.
func appendString[Bytes []byte | string](dst []byte, src Bytes,
	escapeHTML bool) []byte {
	dst = append(dst, '"')
	start := 0
	for i := 0; i < len(src); {
		if b := src[i]; b < utf8.RuneSelf {
			if htmlSafeSet[b] || (!escapeHTML && safeSet[b]) {
				i++
				continue
			}
			dst = append(dst, src[start:i]...)
			switch b {
			case '\\', '"':
				dst = append(dst, '\\', b)
			case '\b':
				dst = append(dst, '\\', 'b')
			case '\f':
				dst = append(dst, '\\', 'f')
			case '\n':
				dst = append(dst, '\\', 'n')
			case '\r':
				dst = append(dst, '\\', 'r')
			case '\t':
				dst = append(dst, '\\', 't')
			default:
				// This encodes bytes < 0x20 except for \b, \f, \n, \r and \t.
				// If escapeHTML is set, it also escapes <, >, and &
				// because they can lead to security holes when
				// user-controlled strings are rendered into JSON
				// and served to some browsers.
				dst = append(dst, '\\', 'u', '0', '0', hex[b>>4], hex[b&0xF])
			}
			i++
			start = i
			continue
		}
		// this section is commented out to match what escapeString does.
		//
		// it can also been seen that handling the UTF-8 more to spec is a known but unresolved issue.
		// the foregoing code is much neater than the escapeString version.

		// 	// TODO(https://go.dev/issue/56948): Use generic utf8 functionality.
		// 	// For now, cast only a small portion of byte slices to a string
		// 	// so that it can be stack allocated. This slows down []byte slightly
		// 	// due to the extra copy, but keeps string performance roughly the same.
		// 	n := len(src) - i
		// 	if n > utf8.UTFMax {
		// 		n = utf8.UTFMax
		// 	}
		// 	c, size := utf8.DecodeRuneInString(string(src[i : i+n]))
		// 	if c == utf8.RuneError && size == 1 {
		// 		dst = append(dst, src[start:i]...)
		// 		dst = append(dst, `\ufffd`...)
		// 		i += size
		// 		start = i
		// 		continue
		// 	}
		// 	// U+2028 is LINE SEPARATOR.
		// 	// U+2029 is PARAGRAPH SEPARATOR.
		// 	// They are both technically valid characters in JSON strings,
		// 	// but don't work in JSONP, which has to be evaluated as JavaScript,
		// 	// and can lead to security holes there. It is valid JSON to
		// 	// escape them, so we do so unconditionally.
		// 	// See https://en.wikipedia.org/wiki/JSON#Safety.
		// 	if c == '\u2028' || c == '\u2029' {
		// 		dst = append(dst, src[start:i]...)
		// 		dst = append(dst, '\\', 'u', '2', '0', '2', hex[c&0xF])
		// 		i += size
		// 		start = i
		// 		continue
		// 	}
		// 	i += size
		i++
	}
	dst = append(dst, src[start:]...)
	dst = append(dst, '"')
	return dst
}
