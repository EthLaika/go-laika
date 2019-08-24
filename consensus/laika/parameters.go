package laika

import (
	"encoding/binary"
	"io"

	"github.com/pkg/errors"
)

// Parameters is the file header of a plot file.
type Parameters struct {
	Address [20]byte
	M       uint8
	N       uint16
	K       uint8
	D       uint32
}

// ParametersSize is the size of the parameters when serialized into a plotfile.
const ParametersSize = 20 + 1 + 2 + 1 + 4

// ReadFromFile reads parameters from a file.
func (p *Parameters) ReadFromFile(file io.Reader) error {
	if _, err := io.ReadFull(file, p.Address[:]); err != nil {
		return errors.Wrap(err, "failed to read address")
	}

	buf := make([]byte, 4)
	if _, err := io.ReadFull(file, buf[:1]); err != nil {
		return errors.Wrap(err, "failed to read chunk size M")
	}
	p.M = uint8(buf[0])

	if _, err := io.ReadFull(file, buf[:2]); err != nil {
		return errors.Wrap(err, "failed to read row size N")
	}
	p.N = binary.LittleEndian.Uint16(buf[:2])

	if _, err := io.ReadFull(file, buf[:1]); err != nil {
		return errors.Wrap(err, "failed to read iteration count K")
	}
	p.K = uint8(buf[0])

	if _, err := io.ReadFull(file, buf); err != nil {
		return errors.Wrap(err, "failed to read difficulty")
	}
	p.D = binary.LittleEndian.Uint32(buf)

	return nil
}

func (p *Parameters) Check() bool {
	return p.M == M && p.N == N && p.K == K && p.D <= K
}
