package laika

import "encoding/binary"

func encUint32(x uint32) []byte {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, x)
	return buf
}

func encUint64(x uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, x)
	return buf
}

func checkDifficulty(row []byte) bool {
	if row == nil {
		return false
	}

	rowhash := digest(row)
	pivot := binary.BigEndian.Uint32(rowhash[:4])
	return pivot < D
}
