package laika

import (
	"golang.org/x/crypto/sha3"

	"github.com/ethereum/go-ethereum/common"
)

type (
	Nonce = uint32
	Chunk = [M]byte
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

	// difficultyZeros is the number of required leading zeros for the RowGen
	// Proof-of-Work
	// Easier for now than checking against an actual big.Int difficulty
	difficultyZeros = 10
)

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
