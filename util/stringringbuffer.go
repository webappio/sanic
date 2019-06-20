package util

//StringRingBuffer is a type which implements a ring buffer for strings
type StringRingBuffer struct {
	buf []string
	curr int
}

//CreateStringRingBuffer creates a StringRingBuffer of a fixed, given size
func CreateStringRingBuffer(size int) StringRingBuffer {
	return StringRingBuffer{
		buf: make([]string, size),
		curr: 0,
	}
}

func (r StringRingBuffer) peek(num int) []string {
	start := r.curr - num + 1
	if start < 0 {
		start += len(r.buf)
	}

	ret := make([]string, 0, num)
	for i := 0; i < num; i++ {
		ret[i] = r.buf[(start + i) % len(r.buf)]
	}
	return ret
}

func (r StringRingBuffer) push(s string) {
	r.buf[r.curr] = s
	r.curr = (r.curr + 1) % len(r.buf)
}