package main

import (
	"flag"
	"log"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/laika"
)

func PlotBlock(file *laika.PlotFile, start uint64, rows uint) {
	block := laika.NewBlock(start, file.Parameters)

	for index := start; index < start+uint64(rows); index++ {
		row, nonce := laika.GenRow(file.Parameters.Address, index)
		block.AddRow(laika.PlotRow{Data: row, Witness: nonce}, file.Parameters)
	}

	if err := file.Plot(block); err != nil {
		panic("failed to plot! OH NO! " + err.Error())
	}
}

func main() {
	size := flag.Uint("size", 1024, "The block size")
	count := flag.Uint("count", 10, "The block count")
	address := flag.String("address", "", "The miner's address")
	file := flag.String("file", "", "The plot file")

	flag.Parse()

	if *address == "" || !common.IsHexAddress(*address) {
		panic("Missing address!")
	}

	if *file == "" {
		panic("Missing file!")
	}

	params := laika.Parameters{
		Address: common.HexToAddress(*address),
		M:       laika.M,
		N:       laika.N,
		K:       laika.K,
		D:       laika.D,
	}

	pfile := laika.OpenOrCreatePlotFile(*file, params.Address)
	if pfile == nil {
		panic("Failed to open plot file!")
	}
	defer pfile.Close()

	for i := uint(0); i < *count; i++ {
		log.Printf("Plotting %d - %d", i**size, i**size+*size)
		PlotBlock(pfile, uint64(i**size), *size)
	}

	log.Println("Done!")
}
