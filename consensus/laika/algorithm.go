package laika

import (
	"math/big"

	"golang.org/x/crypto/sha3"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type (
	Nonce = uint32

	Chunk struct {
		chunk []byte
		idx   uint64
		nonce Nonce
	}

	// ChunkIterator iterates over a column
	ChunkIterator interface {
		Next() bool
		Chunk() Chunk
	}
)

const (
	// M is the byte-size of a chunk
	M = 4

	// N is the number of chunks in a row
	N = 1024

	// L is the byte-size of a row
	L = M * N

	// K is the iteration count in a GenRow run
	K = 4

	// D is the difficulty for row generation
	D uint32 = 0x400000 // 9 leading zeros
)

func GenRow(addr common.Address, idx uint64) (row []byte, nonce Nonce) {
	for ; !checkDifficulty(row); nonce++ {
		row = Row(addr, idx, nonce)
	}
	return
}

func GenProof(h types.Header, ci ChunkIterator) (best Chunk) {
	var bestVal *big.Int

	for ci.Next() {
		c := ci.Chunk()
		phash := proofHash(h, c)
		val := new(big.Int).SetBytes(phash)

		if (bestVal != nil && bestVal.Cmp(val) == 1) || bestVal == nil {
			bestVal = val
			best = c
		}
	}
	return
}

func Row(addr common.Address, idx uint64, nonce Nonce) []byte {
	seed := digest(addr.Bytes(), encUint64(idx), encUint32(nonce))
	// K rows of length L which we iteratively compute
	row := make([]byte, L*K)
	// seed the rows with initial entropy, fully filling the cache
	prng(seed, row)

	for k := 0; k < K; k++ {
		// H(f^k, ..., f^(K-1), g^0, ..., g^(k-1))
		seed := digest(row[k*L:], row[:k*L])
		// Set k'th row g^k
		prng(seed, row[k*L:(k+1)*L])
	}

	// G(idx, nonce) is the row calculated in the last iteration
	return row[(K-1)*L:]
}

func prng(seed, out []byte) {
	sha3.ShakeSum128(out, seed)
}

func digest(data ...[]byte) []byte {
	d := sha3.New256()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(nil)
}
