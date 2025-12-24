package groupcache

import (
	"bytes"
	"errors"
	"io"
	"strings"
)

type ByteView struct {
	b []byte
	s string
}


// Return the length of the v
func (v ByteView) Len() int {
	if v.b != nil {
		return len(v.b)
	}
	return len(v.s)
}

// String changes v to String if v is []byte
func (v ByteView) String() string {
	if v.b != nil {
		return string(v.b)
	}
	return v.s
}

// Copy copies b into dest and returns the number of bytes copied.
func (v ByteView) Copy(dest []byte) int {
	if v.b != nil {
		return copy(dest, v.b)
	}
	return copy(dest, v.s)
}


// Return an io.ReaderSeeker for the bytes in v
func (v ByteView) Reader() io.ReadSeeker {
	// ReaderSeeker 用于从指定的某一点开始传输数据
	// 比如说断点续传 HTTP服务中的Range请求（请求文件的某一部分）
	if v.b != nil {
		return bytes.NewReader(v.b)
	}
	return strings.NewReader(v.s)
}

// WriteTo Implements the io.Write on the bytes in v
func (v ByteView) WriteTo(w io.Writer) (n int64, err error) {
	var m int

	if v.b != nil {
		m, err = w.Write(v.b)
	} else {
		m, err = io.WriteString(w, v.s)
	}

	if err == nil && m < v.Len() {
		err = io.ErrShortWrite
	}

	n = int64(m)
	
	return
}

// ReadAt read data starting from a specified offset into the provided byte slice
// It returns number of bytes and any error encountered.
func (v ByteView) ReadAt(p []byte, off int64) (n int, err error) {
	if off < 0 {
		return 0, errors.New("Invalid offset")
	}

	if off >= int64(v.Len()) {
		return 0, io.EOF
	}

	n = v.SliceFrom(int(off)).Copy(p)
	if n < len(p) {
		err = io.EOF
	}
	return 
}

// EqualBytes returns whether the bytes in view are the same as the b2
func (v ByteView) EqualBytes(b2 []byte) bool {
	if v.b != nil {
		return bytes.Equal(v.b, b2)
	}

	if len(b2) != v.Len() {
		return false
	}

	for i, bi := range b2 {
		if bi != v.s[i] {
			return false
		}
	}
	return true
}

// EqualString returns whether the string in view is the same as the s2
func (v ByteView) EqualString(s2 string) bool {
	if v.b == nil {
		return v.s == s2
	}

	for i, bi := range v.b {
		if bi != s2[i] {
			return false
		}
	}
	return true
}

// Equal returns whether the ByteView is the same as the v2
func (v ByteView) Equal(v2 ByteView) bool {
	if v2.b != nil {
		return v.EqualBytes(v2.b)
	}
	return v.EqualString(v2.s)
}

// SliceFrom slice the view from the provided index to the end
func (v ByteView) SliceFrom(from int) ByteView {
	if v.b != nil {
		return ByteView{b: v.b[from:]}
	}

	return ByteView{s: v.s[from:]}
}

func (v ByteView) Slice(from, to int) ByteView {
	if v.b != nil {
		return ByteView{b: v.b[from:to]}
	}

	return ByteView{s: v.s[from:to]}
}