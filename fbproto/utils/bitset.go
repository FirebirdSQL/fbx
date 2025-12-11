package utils

type BitSet struct {
	bits  []byte
	count int
}

func NewBitSet(count int) *BitSet {
	bytes := make([]byte, int(float64(count+7)/8))
	return &BitSet{bytes, count}
}

func NewBitSetFromBytes(count int, buf []byte) *BitSet {
	return &BitSet{buf, count}
}

func (s *BitSet) Set(i int) {
	b := &s.bits[i/8]
	*b |= byte(1 << (i % 8))
}

func (s *BitSet) Get(i int) bool {
	return s.bits[i/8]&byte(1<<(i%8)) > 0
}

func (s *BitSet) Buf() []byte {
	return s.bits
}

func (s *BitSet) AlignedBuf() []byte {
	padding := (4 - len(s.bits)) & 3
	return append(s.bits, make([]byte, padding)...)
}

func (s *BitSet) Count() int {
	return s.count
}
