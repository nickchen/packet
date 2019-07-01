package packet

// makeMask8 return mask of requested length
func makeMask8(maskLength uint64) uint8 {
	size := uint64(8) - maskLength
	if size >= 0 {
		return uint8(0xff >> size)
	}
	return 0x0
}

// makeMask16 return mask of requested length
func makeMask16(maskLength uint64) uint16 {
	size := uint64(16) - maskLength
	if size >= 0 {
		return uint16(0xffff >> size)
	}
	return 0x0
}
