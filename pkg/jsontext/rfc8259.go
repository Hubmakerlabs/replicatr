package jsontext

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

// EscapeJSONStringAndWrap takes an arbitrary string and escapes all control
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
func EscapeJSONStringAndWrap(s string) (escaped []byte) {
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
			// remaining 8 bit values above 0x20 (Space) are not expanded
		case c >= Space:
			// handle the 6 character escapes \uXXXX remaining.
		case c < Space:
			length += 5
		}
	}
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
	for i := range s {
		c := s[i]
		switch {
		case c == QuotationMark:
			escaped = append(escaped, []byte{ReverseSolidus, QuotationMark}...)
		case c == ReverseSolidus:
			escaped = append(escaped, []byte{ReverseSolidus, ReverseSolidus}...)
		case c >= Space:
			escaped = append(escaped, c)
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
		}
	}
	// add the final double quote character
	escaped = append(escaped, QuotationMark)
	return
}
