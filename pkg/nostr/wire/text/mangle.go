package text

import (
	"io"
)

type Buffer struct {
	Pos int
	Buf []byte
}

// NewBuffer returns a new buffer containing the provided slice. This slice
// can/will be mutated.
func NewBuffer(b []byte) (buf *Buffer) {
	return &Buffer{Buf: b}
}

// Read the next byte out of the buffer or return io.EOF if there is no more.
func (b *Buffer) Read() (bb byte, err error) {
	if b.Pos < len(b.Buf) {
		bb = b.Buf[b.Pos]
		b.Pos++
	} else {
		err = io.EOF
	}
	return
}

// Write a byte into the next index of the buffer or return io.EOF if there is
// no space left.
func (b *Buffer) Write(bb byte) (err error) {
	// log.D.F("writing >%s<", string(bb))
	if b.Pos < len(b.Buf) {
		b.Buf[b.Pos] = bb
		b.Pos++
	} else {
		err = io.EOF
	}
	return
}

// ReadUntil returns all of the buffer from the Pos at invocation, until the
// index immediately before the match of the requested character.
//
// The next Read or Write after this will return the found character or mutate
// it. If the first character at the index of the Pos is the one being
// sought, it returns a zero length slice.
//
// Note that the implementation does not increment the Pos position until
// either the end of the buffer or when the requested character is found,
// because there is no need to write the value twice for no reason.
//
// When this function returns an error, the state of the buffer is unchanged
// from prior to the invocation.
//
// If the character is not `"` then any match within a pair of unescaped `"` is
// ignored. The closing `"` is not counted if it is escaped with a \.
//
// If the character is `"` then any `"` with a `\` before it is ignored (and
// included in the returned slice).
func (b *Buffer) ReadUntil(c byte) (bb []byte, err error) {
	return b.Scan(c, false, true)
}

// ReadThrough is the same as ReadUntil except it returns a slice *including*
// the character being sought.
func (b *Buffer) ReadThrough(c byte) (bb []byte, err error) {
	return b.Scan(c, true, true)
}

// ScanUntil does the same as ReadUntil except it doesn't slice what it passed
// over.
func (b *Buffer) ScanUntil(c byte) (err error) {
	_, err = b.Scan(c, false, false)
	return
}

// ScanThrough does the same as ScanUntil except it returns the next index
// *after* the found item.
func (b *Buffer) ScanThrough(c byte) (err error) {
	_, err = b.Scan(c, true, false)
	return
}

// Scan is the utility back end that does all the scan/read functionality
func (b *Buffer) Scan(c byte, through, slice bool) (subSlice []byte,
	err error) {
	bLen := len(b.Buf)
	// log.D.F("Scan for '%s': %d '%s'", string(c), bLen, string(b.Buf[b.Pos:]))
	var inQuotes bool
	quotes := c == '"'
	for i := b.Pos; i < bLen; i++ {
		// log.D.F("'%s' searching '%s' found: %v, inquotes: %v, through %v quotes %v", string(b.Buf[i]), string(c), c == b.Buf[i], inQuotes, through, quotes)
		// log.D.F("first: quotes: %v %d/%d '%s'", quotes, i, bLen,
		// 	string(b.Buf[i]))
		if !quotes {
			// inQuotes condition only occurs if we aren't searching for a
			// closed quote.
			if !inQuotes {
				// quotes outside of quotes in JSON start quotes, we are not
				// scanning for matches inside quotes.
				if b.Buf[i] == '"' {
					inQuotes = true
					continue
				}
			} else {
				// if we are inside quotes, close them if not escaped.
				if b.Buf[i] == '"' {
					if i > 0 {
						if b.Buf[i] == '\\' {
							continue
						}
					}
					inQuotes = false
					continue
				}
			}
		}
		if b.Buf[i] == c {
			// if we are scanning for inside quotes, match everything except
			// escaped quotes.
			if quotes && i > 0 {
				// quotes with a preceding backslash are ignored
				if b.Buf[i-1] == '\\' {
					continue
				}
			}
			end := i
			if through {
				end++
			}
			if slice {
				subSlice = b.Buf[b.Pos:end]
			}
			// better to set the Pos at the end rather than waste any time
			// mutating two variables when one is enough.
			b.Pos = end
			return
		}
	}
	if slice {
		subSlice = b.Buf[b.Pos:]
	}
	// If we got to the end without a match, set the Pos to the end.
	b.Pos = bLen
	err = io.EOF
	return
}

// ReadEnclosed scans quickly while keeping count of open and close brackets []
// or braces {} and returns the byte sub-slice starting with a bracket and
// ending with the same depth bracket. Selects the counted characters based on the first.
//
// Ignores anything within quotes.
//
// Useful for quickly finding a potentially valid array or object in JSON.
func (b *Buffer) ReadEnclosed() (bb []byte, err error) {
	c := b.Buf[b.Pos]
	bracketed := c == byte('[')
	braced := c == '{'
	if !bracketed && !braced {
		err = log.E.Err("cursor of buffer not on open brace or bracket. found: '%s'",
			string(c))
		return
	}
	var opener, closer byte
	opener, closer = '[', ']'
	if braced {
		opener, closer = '{', '}'
	}
	var depth int
	var inQuotes bool
	for i := b.Pos; i < len(b.Buf); i++ {
		switch b.Buf[i] {
		case '"':
			if inQuotes {
				if i > 0 {
					if b.Buf[i-1] == '\\' {
						// ignore quote if quote is preceded by backslash
						break
					}
				}
				inQuotes = false
			} else {
				inQuotes = true
			}
		case opener:
			if !inQuotes {
				depth++
			}
		case closer:
			if !inQuotes {
				depth--
			}
		}
		if depth == 0 {
			bb = b.Buf[b.Pos : i+1]
			b.Pos = i + 1
			return
		}
	}
	err = io.EOF
	return
}

// ScanForOneOf provides the ability to scan for two or more different bytes.
//
// For simplicity it does not skip quotes, it was actually written to find
// quotes or braces but just to make it clear this is very bare.
//
// if through is set to true, the cursor is advanced to the next after the match
func (b *Buffer) ScanForOneOf(through bool, c ...byte) (which byte, err error) {
	if len(c) < 2 {
		err = log.E.Err("at least two bytes required for ScanUntilOneOf, " +
			"otherwise just use ScanUntil")
		return
	}
	bLen := len(b.Buf)
	for i := b.Pos; i < bLen; i++ {
		for _, d := range c {
			if b.Buf[i] == d {
				which = d
				if through {
					i++
				}
				b.Pos = i
				return
			}
		}
	}
	err = io.EOF
	return
}

// Tail returns the buffer starting from the current Pos position.
func (b *Buffer) Tail() []byte { return b.Buf[b.Pos:] }

// Head returns the buffer from the start until the current Pos position.
func (b *Buffer) Head() []byte { return b.Buf[:b.Pos] }

// WriteBytes copies over top of the current buffer with the bytes given.
//
// Returns io.EOF if the write would exceed the end of the buffer, and does not
// perform the operation, nor move the cursor.
func (b *Buffer) WriteBytes(bb []byte) (err error) {
	if len(bb) == 0 {
		return
	}
	// log.T.F("buf len: %d, pos: %d writing %d '%s'", len(b.Buf), b.Pos, len(bb),
	// 	string(bb))
	until := b.Pos + len(bb)
	if until <= len(b.Buf) {
		copy(b.Buf[b.Pos:until], bb)
		b.Pos = until
	} else {
		err = io.EOF
	}
	return
}

// ReadBytes returns the specified number of byte, and advances the cursor, or
// io.EOF if there isn't this much remaining after the cursor.
func (b *Buffer) ReadBytes(count int) (bb []byte, err error) {
	until := b.Pos + count
	if until < len(b.Buf) {
		bb = b.Buf[b.Pos:until]
		b.Pos = until
	} else {
		err = io.EOF
	}
	return
}

// Copy a given length of bytes starting at src position to dest position, and
// move the cursor to the end of the written segment.
func (b *Buffer) Copy(length, src, dest int) (err error) {
	// Zero length is a no-op.
	if length == 0 {
		return
	}
	// if nothing would be copied, just update the cursor.
	if src == dest {
		b.Pos = src + length
		return
	}
	bLen := len(b.Buf)
	// if the length is negative or the offset from src or dest to the length
	// exceeds the size of the slice, return io.EOF to signify the operation
	// would exceed the bounds of the slice.
	if length < 0 || src+length >= bLen || dest+length >= bLen {
		return io.EOF
	}
	// copy the src segment over top of the dest segment. Note that Go
	// automatically switches the operation order from lower->higher or higher->
	// lower if the segments overlap, so that the source is not mutated before
	// the destination.
	//
	// update the cursor first to use in the copy operation being the new cursor
	// position and right-most index immediately after the last byte written
	// over.
	b.Pos = dest + length
	copy(b.Buf[dest:b.Pos], b.Buf[src:src+length])
	return
}

// String returns the whole buffer as a string.
func (b *Buffer) String() (s string) { return string(b.Buf) }
func (b *Buffer) Bytes() (bb []byte) { return b.Buf }
