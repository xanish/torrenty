package bitfield

type Bitfield []byte

func (bf Bitfield) HasPiece(index uint32) bool {
	byteIndex := index / 8
	offset := index % 8

	if byteIndex < 0 || byteIndex >= uint32(len(bf)) {
		return false
	}

	return bf[byteIndex]>>uint(7-offset)&1 != 0
}

func (bf Bitfield) SetPiece(index uint32) {
	byteIndex := index / 8
	offset := index % 8

	if byteIndex < 0 || byteIndex >= uint32(len(bf)) {
		return
	}

	bf[byteIndex] |= 1 << uint(7-offset)
}
