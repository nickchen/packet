package packet

func makeMask(maskLength uint) uint64 {
	if 64 >= maskLength {
		return 0xffffffffffffffff >> (64 - maskLength)
	}
	return 0x0
}
