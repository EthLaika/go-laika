package laika

import (
	"log"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type memChunkIt struct {
	col    int
	nidx   int
	rows   [][]byte
	nonces []Nonce
}

func (it *memChunkIt) Next() bool {
	it.nidx++
	return (it.nidx <= len(it.rows))
}

func (it *memChunkIt) Chunk() Chunk {
	return Chunk{
		chunk: it.rows[it.nidx-1][it.col*M : (it.col+1)*M],
		idx:   uint64(it.nidx - 1),
		nonce: it.nonces[it.nidx-1],
	}
}

func TestGenProofVrfy(t *testing.T) {
	var addr common.Address

	I := uint64(2) // number of rows
	rows := make([][]byte, I)
	nonces := make([]Nonce, I)

	for i := uint64(0); i < I; i++ {
		rows[i], nonces[i] = GenRow(addr, i)
	}

	header := &types.Header{
		Coinbase: addr,
		Number:   new(big.Int),
	}

	ci := &memChunkIt{
		col:    challengeCol(headerHash(header)),
		rows:   rows,
		nonces: nonces,
	}

	log.Print("challenge col:", ci.col)

	GenProof(header, ci)

	log.Print("header proof:", chunkFromHeader(header))

	if err := VrfyProof(header); err != nil {
		t.Error("proof should verify:", err)
	}
}
