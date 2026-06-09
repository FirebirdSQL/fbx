package fbproto

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBitSet(t *testing.T) {
	assert.Equal(t, 1, len(NewBitSet(1).bits))
	assert.Equal(t, 1, len(NewBitSet(8).bits))
	assert.Equal(t, 2, len(NewBitSet(9).bits))
	assert.Equal(t, 2, len(NewBitSet(16).bits))
}

func TestNewBitSet_Set(t *testing.T) {
	bs := NewBitSet(16)
	bs.Set(0)
	assert.Equal(t, "[00000001 00000000]", fmt.Sprintf("%08b", bs.bits))
	bs.Set(1)
	assert.Equal(t, "[00000011 00000000]", fmt.Sprintf("%08b", bs.bits))
	bs.Set(7)
	assert.Equal(t, "[10000011 00000000]", fmt.Sprintf("%08b", bs.bits))
	bs.Set(8)
	assert.Equal(t, "[10000011 00000001]", fmt.Sprintf("%08b", bs.bits))
	bs.Set(14)
	assert.Equal(t, "[10000011 01000001]", fmt.Sprintf("%08b", bs.bits))
	bs.Set(15)
	assert.Equal(t, "[10000011 11000001]", fmt.Sprintf("%08b", bs.bits))
}

func TestNewBitSet_Get(t *testing.T) {
	bs := NewBitSetFromBytes(16, []byte{0x0F, 0x10})
	assert.Equal(t, "[00001111 00010000]", fmt.Sprintf("%08b", bs.bits))
	assert.Equal(t, true, bs.Get(0))
	assert.Equal(t, true, bs.Get(1))
	assert.Equal(t, true, bs.Get(2))
	assert.Equal(t, true, bs.Get(3))
	assert.Equal(t, false, bs.Get(4))
	assert.Equal(t, false, bs.Get(5))
	assert.Equal(t, false, bs.Get(6))
	assert.Equal(t, false, bs.Get(7))
	assert.Equal(t, false, bs.Get(11))
	assert.Equal(t, true, bs.Get(12))
	assert.Equal(t, false, bs.Get(13))
}

func TestBuf(t *testing.T) {
	bs := NewBitSet(1)
	bs.Set(0)
	assert.Equal(t, "[00000001]", fmt.Sprintf("%08b", bs.Buf()))
}

func TestAlignedBuf(t *testing.T) {
	bs := NewBitSet(1)
	bs.Set(0)
	assert.Equal(t, "[00000001 00000000 00000000 00000000]", fmt.Sprintf("%08b", bs.AlignedBuf()))
	bs = NewBitSet(9)
	bs.Set(1)
	bs.Set(8)
	assert.Equal(t, "[00000010 00000001 00000000 00000000]", fmt.Sprintf("%08b", bs.AlignedBuf()))
	bs = NewBitSet(18)
	bs.Set(2)
	bs.Set(8)
	bs.Set(17)
	assert.Equal(t, "[00000100 00000001 00000010 00000000]", fmt.Sprintf("%08b", bs.AlignedBuf()))
	bs = NewBitSet(32)
	bs.Set(3)
	bs.Set(10)
	bs.Set(20)
	bs.Set(30)
	assert.Equal(t, "[00001000 00000100 00010000 01000000]", fmt.Sprintf("%08b", bs.AlignedBuf()))
	bs = NewBitSet(33)
	bs.Set(32)
	assert.Equal(t, "[00000000 00000000 00000000 00000000 00000001 00000000 00000000 00000000]", fmt.Sprintf("%08b", bs.AlignedBuf()))
}
