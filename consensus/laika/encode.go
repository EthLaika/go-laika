package laika

import (
	"encoding/binary"
	"io"

	"golang.org/x/crypto/sha3"

	"github.com/ethereum/go-ethereum/core/types"
)

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

func headerHash(h *types.Header) []byte {
	digester := sha3.New256()
	headerEncode(digester, h)
	return digester.Sum(nil)
}

func headerEncode(io.Writer, *types.Header)

func proofHash(header *types.Header, chunk Chunk) []byte {
	return digest(
		header.Coinbase[:],
		headerHash(header),
		chunk.chunk,
		encUint64(chunk.idx),
		encUint32(chunk.nonce),
	)
}

func chunkFromHeader(h *types.Header) Chunk {
	return Chunk{
		chunk: h.LaikaChunk,
		idx:   h.LaikaIdx,
		nonce: binary.LittleEndian.Uint32(h.Nonce[:4]),
	}
}
