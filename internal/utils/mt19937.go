package utils

const (
	mtN       = 624
	mtM       = 397
	matrixA   = 0x9908b0df
	upperMask = 0x80000000
	lowerMask = 0x7fffffff
)

type MT19937 struct {
	state [mtN]uint32
	index int
}

func NewMT19937(seed uint32) *MT19937 {
	mt := &MT19937{index: mtN}
	mt.state[0] = seed
	for i := 1; i < mtN; i++ {
		prev := mt.state[i-1]
		mt.state[i] = 1812433253*(prev^(prev>>30)) + uint32(i)
	}
	return mt
}

func (mt *MT19937) Uint32() uint32 {
	if mt.index >= mtN {
		mt.twist()
	}
	y := mt.state[mt.index]
	mt.index++
	// tempering
	y ^= y >> 11
	y ^= (y << 7) & 0x9d2c5680
	y ^= (y << 15) & 0xefc60000
	y ^= y >> 18
	return y
}

func (mt *MT19937) twist() {
	for i := 0; i < mtN; i++ {
		y := (mt.state[i] & upperMask) | (mt.state[(i+1)%mtN] & lowerMask)
		mt.state[i] = mt.state[(i+mtM)%mtN] ^ (y >> 1)
		if y%2 != 0 {
			mt.state[i] ^= matrixA
		}
	}
	mt.index = 0
}
