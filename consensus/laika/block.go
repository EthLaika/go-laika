package laika

import (
	"encoding/binary"
	"io"

	"github.com/pkg/errors"
)

// BlockHeader contains the dimensions of a plotted block.
type BlockHeader struct {
	Start  uint64
	Length uint16
}

// Block is a block to be plotted.
type Block struct {
	BlockHeader
	Columns   [][]byte
	Witnesses []uint32
}

// PlotRow is a generated row and the witness.
type PlotRow struct {
	Data    []byte
	Witness uint32
}

func (r *PlotRow) Chunk(i int, p Parameters) []byte {
	start := i * int(p.M)
	return r.Data[start : start+int(p.M)]
}

// NewBlock creates a new block with the specified start nonce.
// The block is initially empty.
func NewBlock(start uint64, p Parameters) Block {
	return Block{
		BlockHeader: BlockHeader{
			Start:  start,
			Length: 0,
		},
		Columns:   make([][]byte, p.N),
		Witnesses: nil,
	}
}

// Read reads a block header from a file.
func (b *BlockHeader) Read(file io.Reader) (err error) {
	buf := make([]byte, 8)
	if _, err = io.ReadFull(file, buf); err != nil {
		return errors.Wrap(err, "failed to read block start index")
	}
	b.Start = binary.LittleEndian.Uint64(buf)

	if _, err = io.ReadFull(file, buf[:2]); err != nil {
		return errors.Wrap(err, "failed to read block length")
	}
	b.Length = binary.LittleEndian.Uint16(buf[:2])

	return nil
}

// FreeRows returns the number of rows that can still be added to a block.
func (b Block) FreeRows() int {
	return (1 << 16) - int(b.Length)
}

// AddRows adds an array of rows to a block.
// Panics if you try to add more rows than returned by FreeRows().
func (b *Block) AddRows(rows []PlotRow, p Parameters) {
	if b.FreeRows() < len(rows) {
		panic("Tried to add more rows than possible")
	}
	b.Length += uint16(len(rows))
	for _, row := range rows {
		for i := 0; i < int(p.N); i++ {
			b.Columns[i] = append(b.Columns[i], row.Chunk(i, p)...)
		}
	}
}

// AddRow adds a single row to a block.
// Panics if you try to add more rows than returned by FreeRows().
func (b *Block) AddRow(row PlotRow, p Parameters) {
	if b.FreeRows() == 0 {
		panic("Tried to add more rows than possible")
	}
	b.Length++
	for i := 0; i < int(p.N); i++ {
		b.Columns[i] = append(b.Columns[i], row.Chunk(i, p)...)
	}
	b.Witnesses = append(b.Witnesses, row.Witness)
}

// Append appends a block to a file.
func (b *Block) Append(f io.Writer) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, b.Start)
	if _, err := f.Write(buf); err != nil {
		return errors.Wrap(err, "failed to write the block start index")
	}

	binary.LittleEndian.PutUint16(buf[:2], b.Length)
	if _, err := f.Write(buf[:2]); err != nil {
		return errors.Wrap(err, "failed to write the block length")
	}

	for _, col := range b.Columns {
		if _, err := f.Write(col); err != nil {
			return errors.Wrap(err, "failed to write the columns")
		}
	}

	for _, w := range b.Witnesses {
		binary.LittleEndian.PutUint32(buf[:4], w)
		if _, err := f.Write(buf[:4]); err != nil {
			return errors.Wrap(err, "failed to write witness")
		}
	}

	return nil
}
