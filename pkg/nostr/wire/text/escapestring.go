package text

// EscapeString for JSON encoding according to RFC8259.
//
// taken from https://github.com/nbd-wtf/go-nostr/blob/master/utils.go replaced
// by EscapeJSONStringAndWrap in file rfc8259.go tested to be functionally
// equivalent, the purpose of the above function is to eliminate extra heap
// allocations for very long strings such as long form posts.
//
// Formatting is retained from the original despite being ugly.
func EscapeString(dst []byte, s string) []byte {
	dst = append(dst, '"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"':
			// quotation mark
			dst = append(dst, []byte{'\\', '"'}...)
		case c == '\\':
			// reverse solidus
			dst = append(dst, []byte{'\\', '\\'}...)
		case c >= 0x20:
			// default, rest below are control chars
			dst = append(dst, c)
		case c == 0x08:
			dst = append(dst,
				[]byte{'\\', 'b'}...)
		case c < 0x09:
			dst = append(dst,
				[]byte{'\\', 'u', '0', '0', '0', '0' + c}...)
		case c == 0x09:
			dst = append(dst,
				[]byte{'\\', 't'}...)
		case c == 0x0a:
			dst = append(dst,
				[]byte{'\\', 'n'}...)
		case c == 0x0c:
			dst = append(dst,
				[]byte{'\\', 'f'}...)
		case c == 0x0d:
			dst = append(dst,
				[]byte{'\\', 'r'}...)
		case c < 0x10:
			dst = append(dst,
				[]byte{'\\', 'u', '0', '0', '0', 0x57 + c}...)
		case c < 0x1a:
			dst = append(dst,
				[]byte{'\\', 'u', '0', '0', '1', 0x20 + c}...)
		case c < 0x20:
			dst = append(dst,
				[]byte{'\\', 'u', '0', '0', '1', 0x47 + c}...)
		}
	}
	dst = append(dst, '"')
	return dst
}
