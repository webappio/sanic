package util

//StringRingBuffer is a type which implements a ring buffer for strings
type StringRingBuffer struct {
	buf   []string
	curr  int
	usage int
}

//CreateStringRingBuffer creates a StringRingBuffer of a fixed, given size
func CreateStringRingBuffer(size int) *StringRingBuffer {
	return &StringRingBuffer{
		buf:  make([]string, size),
		curr: 0,
	}
}

//Peek gets the last N elements inserted into this buffer, where the latest is the last element returned
func (r *StringRingBuffer) Peek(num int) []string {
	start := r.curr - num
	for start < 0 {
		start += len(r.buf)
	}

	ret := make([]string, num)
	for i := 0; i < num && i < len(r.buf); i++ {
		ret[i] = r.buf[(start + i) % len(r.buf)]
	}
	return ret
}

//Push adds a new element to this ring buffer
func (r *StringRingBuffer) Push(s string) {
	r.buf[r.curr] = s
	r.curr = (r.curr + 1) % len(r.buf)
	if r.usage < len(r.buf) {
		r.usage += 1
	}
}

//Usage returns the number of elements, up to capacity, which have been inserted into this ring buffer.
func (r *StringRingBuffer) Usage() int {
	return r.usage
}
