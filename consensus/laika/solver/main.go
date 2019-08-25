package main

import (
	"flag"
	"log"
	"math/big"

	"golang.org/x/crypto/sha3"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/laika"
	"github.com/ethereum/go-ethereum/core/types"
)

func main() {
	file := flag.String("file", "", "The plot file")
	challenge := flag.String("challenge", "", "The challenge string")

	flag.Parse()

	if *file == "" {
		panic("Missing file!")
	}

	chHash := sha3.Sum256([]byte(*challenge))

	header := &types.Header{
		Number: new(big.Int),
		Extra:  chHash[:],
	}

	pfile := laika.OpenPlotFile(*file)
	if pfile == nil {
		panic("Failed to open plot file!")
	}
	defer pfile.Close()

	log.Println("Challenge: ", hexutil.Encode(chHash[:]))
	laika.GenProof(header, pfile.Iterator(laika.ChallengeCol(chHash[:])))
	log.Println("Hash: ", hexutil.Encode(laika.ProofHash(header)))
}
