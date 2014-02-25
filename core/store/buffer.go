package store

import (
	"errors"
	"fmt"
	// "github.com/balzaczyy/golucene/core/util"
)

type SeekReader interface {
	seekInternal(pos int64) error
	readInternal(buf []byte) error
}

type BufferedIndexInput struct {
	*IndexInputImpl
	SeekReader
	bufferSize     int
	buffer         []byte
	bufferStart    int64
	bufferLength   int
	bufferPosition int
}

func newBufferedIndexInput(desc string, context IOContext) *BufferedIndexInput {
	return newBufferedIndexInputBySize(desc, bufferSize(context))
}

func newBufferedIndexInputBySize(desc string, bufferSize int) *BufferedIndexInput {
	checkBufferSize(bufferSize)
	ans := &BufferedIndexInput{bufferSize: bufferSize}
	ans.IndexInputImpl = newIndexInputImpl(desc, ans)
	return ans
}

func (in *BufferedIndexInput) ReadByte() (b byte, err error) {
	if in.bufferPosition >= in.bufferLength {
		in.refill()
	}
	in.bufferPosition++
	return in.buffer[in.bufferPosition-1], nil
}

func (in *BufferedIndexInput) newBuffer(newBuffer []byte) {
	// Subclasses can do something here
	in.buffer = newBuffer
}

func checkBufferSize(bufferSize int) {
	if bufferSize <= 0 {
		panic(fmt.Sprintf("bufferSize must be greater than 0 (got %v)", bufferSize))
	}
}

func (in *BufferedIndexInput) ReadBytes(buf []byte) error {
	return in.ReadBytesBuffered(buf, true)
}

func (in *BufferedIndexInput) ReadBytesBuffered(buf []byte, useBuffer bool) error {
	available := in.bufferLength - in.bufferPosition
	if length := len(buf); length <= available {
		// the buffer contains enough data to satisfy this request
		if length > 0 { // to allow b to be null if len is 0...
			copy(buf, in.buffer[in.bufferPosition:in.bufferPosition+length])
		}
		in.bufferPosition += length
	} else {
		// the buffer does not have enough data. First serve all we've got.
		if available > 0 {
			copy(buf, in.buffer[in.bufferPosition:in.bufferPosition+available])
			buf = buf[available:]
			in.bufferPosition += available
		}
		// and now, read the remaining 'len' bytes:
		if length := len(buf); useBuffer && length < in.bufferSize {
			// If the amount left to read is small enough, and
			// we are allowed to use our buffer, do it in the usual
			// buffered way: fill the buffer and copy from it:
			if err := in.refill(); err != nil {
				return err
			}
			if in.bufferLength < length {
				// Throw an exception when refill() could not read len bytes:
				copy(buf, in.buffer[0:in.bufferLength])
				return errors.New(fmt.Sprintf("read past EOF: %v", in))
			} else {
				copy(buf, in.buffer[0:length])
				in.bufferPosition += length
			}
		} else {
			// The amount left to read is larger than the buffer
			// or we've been asked to not use our buffer -
			// there's no performance reason not to read it all
			// at once. Note that unlike the previous code of
			// this function, there is no need to do a seek
			// here, because there's no need to reread what we
			// had in the buffer.
			length := len(buf)
			after := in.bufferStart + int64(in.bufferPosition) + int64(length)
			if after > in.Length() {
				return errors.New(fmt.Sprintf("read past EOF: %v", in))
			}
			if err := in.readInternal(buf); err != nil {
				return err
			}
			in.bufferStart = after
			in.bufferPosition = 0
			in.bufferLength = 0 // trigger refill() on read
		}
	}
	return nil
}

func (in *BufferedIndexInput) ReadShort() (n int16, err error) {
	if 2 <= in.bufferLength-in.bufferPosition {
		in.bufferPosition += 2
		return (int16(in.buffer[in.bufferPosition-2]) << 8) | int16(in.buffer[in.bufferPosition-1]), nil
	}
	return in.DataInputImpl.ReadShort()
}

func (in *BufferedIndexInput) ReadInt() (n int32, err error) {
	if 4 <= in.bufferLength-in.bufferPosition {
		// log.Print("Reading int from buffer...")
		in.bufferPosition += 4
		return (int32(in.buffer[in.bufferPosition-4]) << 24) | (int32(in.buffer[in.bufferPosition-3]) << 16) |
			(int32(in.buffer[in.bufferPosition-2]) << 8) | int32(in.buffer[in.bufferPosition-1]), nil
	}
	return in.DataInputImpl.ReadInt()
}

func (in *BufferedIndexInput) ReadLong() (n int64, err error) {
	if 8 <= in.bufferLength-in.bufferPosition {
		in.bufferPosition += 4
		i1 := (int64(in.buffer[in.bufferPosition-4]) << 24) | (int64(in.buffer[in.bufferPosition-3]) << 16) |
			(int64(in.buffer[in.bufferPosition-2]) << 8) | int64(in.buffer[in.bufferPosition-1])
		in.bufferPosition += 4
		i2 := (int64(in.buffer[in.bufferPosition-4]) << 24) | (int64(in.buffer[in.bufferPosition-3]) << 16) |
			(int64(in.buffer[in.bufferPosition-2]) << 8) | int64(in.buffer[in.bufferPosition-1])
		return (i1 << 32) | i2, nil
	}
	return in.DataInputImpl.ReadLong()
}

func (in *BufferedIndexInput) ReadVInt() (n int32, err error) {
	if 5 <= in.bufferLength-in.bufferPosition {
		b := in.buffer[in.bufferPosition]
		in.bufferPosition++
		if b < 128 {
			return int32(b), nil
		}
		n := int32(b) & 0x7F
		b = in.buffer[in.bufferPosition]
		in.bufferPosition++
		n |= (int32(b) & 0x7F) << 7
		if b < 128 {
			return n, nil
		}
		b = in.buffer[in.bufferPosition]
		in.bufferPosition++
		n |= (int32(b) & 0x7F) << 14
		if b < 128 {
			return n, nil
		}
		b = in.buffer[in.bufferPosition]
		in.bufferPosition++
		n |= (int32(b) & 0x7F) << 21
		if b < 128 {
			return n, nil
		}
		b = in.buffer[in.bufferPosition]
		in.bufferPosition++
		// Warning: the next ands use 0x0F / 0xF0 - beware copy/paste errors:
		n |= (int32(b) & 0x0F) << 28
		if (b & 0xF0) == 0 {
			return n, nil
		}
		return 0, errors.New("Invalid vInt detected (too many bits)")
	}
	return in.DataInputImpl.ReadVInt()
}

func (in *BufferedIndexInput) ReadVLong() (n int64, err error) {
	if 9 <= in.bufferLength-in.bufferPosition {
		b := in.buffer[in.bufferPosition]
		in.bufferPosition++
		if b < 128 {
			return int64(b), nil
		}
		n := int64(b & 0x7F)
		b = in.buffer[in.bufferPosition]
		in.bufferPosition++
		n |= (int64(b&0x7F) << 7)
		if b < 128 {
			return n, nil
		}
		b = in.buffer[in.bufferPosition]
		in.bufferPosition++
		n |= (int64(b&0x7F) << 14)
		if b < 128 {
			return n, nil
		}
		b = in.buffer[in.bufferPosition]
		in.bufferPosition++
		n |= (int64(b&0x7F) << 21)
		if b < 128 {
			return n, nil
		}
		b = in.buffer[in.bufferPosition]
		in.bufferPosition++
		n |= (int64(b&0x7F) << 28)
		if b < 128 {
			return n, nil
		}
		b = in.buffer[in.bufferPosition]
		in.bufferPosition++
		n |= (int64(b&0x7F) << 35)
		if b < 128 {
			return n, nil
		}
		b = in.buffer[in.bufferPosition]
		in.bufferPosition++
		n |= (int64(b&0x7F) << 42)
		if b < 128 {
			return n, nil
		}
		b = in.buffer[in.bufferPosition]
		in.bufferPosition++
		n |= (int64(b&0x7F) << 49)
		if b < 128 {
			return n, nil
		}
		b = in.buffer[in.bufferPosition]
		in.bufferPosition++
		n |= (int64(b&0x7F) << 56)
		if b < 128 {
			return n, nil
		}
		return 0, errors.New("Invalid vLong detected (negative values disallowed)")
	}
	return in.DataInputImpl.ReadVLong()
}

// use panic/recover to handle error
func (in *BufferedIndexInput) refill() error {
	start := in.bufferStart + int64(in.bufferPosition)
	end := start + int64(in.bufferSize)
	if end > in.Length() { // don't read past EOF
		end = in.Length()
	}
	newLength := int(end - start)
	if newLength <= 0 {
		return errors.New(fmt.Sprintf("read past EOF: %v", in))
	}

	if in.buffer == nil {
		in.newBuffer(make([]byte, in.bufferSize)) // allocate buffer lazily
		in.seekInternal(int64(in.bufferStart))
	}
	in.readInternal(in.buffer[0:newLength])
	in.bufferLength = newLength
	in.bufferStart = start
	in.bufferPosition = 0
	return nil
}

func (in *BufferedIndexInput) FilePointer() int64 {
	return in.bufferStart + int64(in.bufferPosition)
}

func (in *BufferedIndexInput) Seek(pos int64) error {
	if pos >= in.bufferStart && pos < in.bufferStart+int64(in.bufferLength) {
		in.bufferPosition = int(pos - in.bufferStart) // seek within buffer
		return nil
	} else {
		in.bufferStart = pos
		in.bufferPosition = 0
		in.bufferLength = 0 // trigger refill() on read()
		return in.seekInternal(pos)
	}
}

// type BufferedIndexInput struct {
// 	*IndexInputImpl
// 	bufferSize     int
// 	buffer         []byte
// 	bufferStart    int64
// 	bufferLength   int
// 	bufferPosition int
// 	seekInternal   func(pos int64)
// 	readInternal   func(buf []byte) error
// }

func (in *BufferedIndexInput) Clone() IndexInput {
	ans := &BufferedIndexInput{
		bufferSize:     in.bufferSize,
		buffer:         nil,
		bufferStart:    in.FilePointer(),
		bufferLength:   0,
		bufferPosition: 0,
	}
	ans.IndexInputImpl = newIndexInputImpl(in.desc, ans)
	return ans
}

type flushBufferAndLength interface {
	flushBuffer(b []byte) error
	Length() int64
}

type BufferedIndexOutput struct {
	*IndexOutputImpl
	flushBufferAndLength
	buffer   []byte
	start    int64 // position in file of buffer
	position int   // position in buffer
}

/*
Creates a new BufferedIndexOutput with the given buffer size.
*/
func newBufferedIndexOutput(size int, part flushBufferAndLength) *BufferedIndexOutput {
	assert2(size > 0, fmt.Sprintf("bufferSize must be greater than 0 (got %v)", size))
	out := &BufferedIndexOutput{flushBufferAndLength: part, buffer: make([]byte, size)}
	out.IndexOutputImpl = newIndexOutput(out)
	return out
}

func (out *BufferedIndexOutput) flush() error {
	err := out.flushBuffer(out.buffer[:out.position])
	if err == nil {
		out.start += int64(out.position)
		out.position = 0
	}
	return err
}

func (out BufferedIndexOutput) Close() error {
	return out.flush()
}
