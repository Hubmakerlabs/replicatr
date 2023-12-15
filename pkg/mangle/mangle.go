// Package mangle is a simplified version of mangle.Buffer that is specifically
// designed for mutating, not growing the length of the buffer, thus the name
// mangle.
//

package mangle

import "io"

type Buffer struct {
	Pos int
	Buf []byte
}

// New returns a new buffer containing the provided slice. This slice
// can/will be mutated.
func New(b []byte) (buf *Buffer) {
	return &Buffer{Buf: b}
}

// Read the next byte out of the buffer or return io.EOF if there is no more.
func (b *Buffer) Read() (bb byte, e error) {
	if b.Pos < len(b.Buf) {
		bb = b.Buf[b.Pos]
		b.Pos++
	} else {
		e = io.EOF
	}
	return
}

// Write a byte into the next index of the buffer or return io.EOF if there is
// no space left.
func (b *Buffer) Write(bb byte) (e error) {
	if b.Pos < len(b.Buf) {
		b.Buf[b.Pos] = bb
		b.Pos++
	} else {
		e = io.EOF
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
func (b *Buffer) ReadUntil(c byte) (bb []byte, e error) {
	bLen := len(b.Buf)
	for i := b.Pos; i < bLen; i++ {
		if b.Buf[i] == c {
			bb = b.Buf[b.Pos:i]
			// better to set the Pos at the end rather than waste any time
			// mutating two variables when one is enough.
			b.Pos = i
			return
		}
	}
	// If we got to the end without a match, set the Pos to the end.
	b.Pos = bLen
	return nil, io.EOF
}

// ReadThrough is the same as ReadUntil except it returns a slice *including*
// the character being sought.
func (b *Buffer) ReadThrough(c byte) (bb []byte, e error) {
	bLen := len(b.Buf)
	for i := b.Pos; i < bLen; i++ {
		if b.Buf[i] == c {
			end := i + 1
			bb = b.Buf[b.Pos:end]
			b.Pos = end
			return
		}
	}
	// If we got to the end without a match, set the Pos to the end.
	b.Pos = bLen
	return nil, io.EOF
}

// ScanUntil does the same as ReadUntil except it doesn't slice what it passed
// over.
func (b *Buffer) ScanUntil(c byte) (e error) {
	bLen := len(b.Buf)
	for i := b.Pos; i < bLen; i++ {
		if b.Buf[i] == c {
			// better to set the Pos at the end rather than waste any time
			// mutating two variables when one is enough.
			b.Pos = i
			return
		}
	}
	// If we got to the end without a match, set the Pos to the end.
	b.Pos = bLen
	return io.EOF
}

// ScanThrough does the same as ScanUntil except it returns the next index
// *after* the found item.
func (b *Buffer) ScanThrough(c byte) (e error) {
	bLen := len(b.Buf)
	for i := b.Pos; i < bLen; i++ {
		if b.Buf[i] == c {
			// better to set the Pos at the end rather than waste any time
			// mutating two variables when one is enough.
			b.Pos = i + 1
			return
		}
	}
	// If we got to the end without a match, set the Pos to the end.
	b.Pos = bLen
	return io.EOF
}

// Tail returns the buffer starting from the current Pos position.
func (b *Buffer) Tail() []byte { return b.Buf[b.Pos:] }

// Head returns the buffer from the start until the current Pos position.
func (b *Buffer) Head() []byte { return b.Buf[:b.Pos] }

// WriteBytes copies over top of the current buffer with the bytes given.
//
// Returns io.EOF if the write would exceed the end of the buffer, and does not
// perform the operation, nor move the cursor.
func (b *Buffer) WriteBytes(bb []byte) (e error) {
	if len(bb) == 0 {
		return
	}
	until := b.Pos + len(bb)
	if until < len(b.Buf) {
		copy(b.Buf[b.Pos:until], bb)
		b.Pos = until
	} else {
		e = io.EOF
	}
	return
}

// ReadBytes returns the specified number of byte, and advances the cursor, or
// io.EOF if there isn't this much remaining after the cursor.
func (b *Buffer) ReadBytes(count int) (bb []byte, e error) {
	until := b.Pos + count
	if until < len(b.Buf) {
		bb = b.Buf[b.Pos:until]
		b.Pos = until
	} else {
		e = io.EOF
	}
	return
}

// Copy a given length of bytes starting at src position to dest position, and
// move the cursor to the end of the written segment.
func (b *Buffer) Copy(length, src, dest int) (e error) {
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

func (b *Buffer) String() (s string) { return string(b.Buf) }
func (b *Buffer) Bytes() (bb []byte) { return b.Buf }
