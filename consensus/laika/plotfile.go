package laika

import (
	"encoding/binary"
	"io"
	"os"
	"sync"

	"github.com/pkg/errors"
)

// PlotFile is a handle to a laika plot file.
// It is used to mine and to plot.
type PlotFile struct {
	file       *os.File
	parameters Parameters
	mutex      sync.Mutex
	used       bool
}

type ColumnBlock struct {
	BlockHeader
	chunks    []byte
	witnesses []uint32
}

// OpenPlotFile opens a plot file for both reading and writing.
func OpenPlotFile(file string) *PlotFile {
	handle, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		return nil
	}
	f := &PlotFile{
		file: handle,
	}

	if err := f.parameters.ReadFromFile(f.file); err != nil || !f.parameters.Check() {
		handle.Close()
		return nil
	}

	return f
}

func (f *PlotFile) Close() error {
	return f.file.Close()
}

// reset seeks a plot file to the very beginning of the file.
func (f *PlotFile) reset() {
	f.file.Seek(ParametersSize, 0)
}

// Creates a chunk iterator for a file.
// Only one chunk iterator per file may exist at a time.
// The chunk iterator has to be manually destroyed if it is not completely used up.
func (f *PlotFile) Iterator() (it ChunkIterator) {
	// Check for concurrent iterators.
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if f.used {
		panic("plotfile.Iterator: concurrency detected")
	}
	f.used = true

	it.file = f
	return
}

// ReadOneBlock reads all of a block's chunks within the selected column from a plot file.
// After a successful operation, the file is seeked to the beginning of the next block.
func (f *PlotFile) ReadOneBlock(column int) (cols ColumnBlock, err error) {
	if err := cols.Read(f.file); err != nil {
		return ColumnBlock{}, errors.WithMessage(err, "failed to read the block header")
	}

	// Skip the first columns.
	f.file.Seek(int64(column)*int64(f.parameters.M)*int64(cols.Length), 1)

	// Read the chunks.
	cols.chunks = make([]byte, int(f.parameters.M)*int(cols.Length))
	if _, err := io.ReadFull(f.file, cols.chunks); err != nil {
		return ColumnBlock{}, errors.Wrap(err, "failed to read the chunks")
	}

	// Seek to the witnesses.
	f.file.Seek(
		int64(f.parameters.M)*
			int64(cols.Length)*
			(int64(f.parameters.N)-int64(column)),
		1)

	// Read the witnesses.
	cols.witnesses = make([]uint32, cols.Length)
	buf := make([]byte, 4)
	for i := range cols.chunks {
		if _, err := io.ReadFull(f.file, buf); err != nil {
			return ColumnBlock{}, errors.New("failed to read the witness")
		}
		cols.witnesses[i] = binary.LittleEndian.Uint32(buf)
	}

	return
}

// Plots one block at the end of the plot file.
func (f *PlotFile) Plot(b Block) error {
	return b.Append(f.file)
}
