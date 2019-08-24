package laika

import (
	"encoding/binary"
	"io"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/common"
)

// Parameters is the file header of a plot file.
type Parameters struct {
	Address common.Address
	M       uint8
	N       uint16
	K       uint8
	D       uint32
}

// ParametersSize is the size of the parameters when serialized into a plotfile.
const ParametersSize = common.AddressLength + 1 + 2 + 1 + 4

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

func (p *Parameters) WriteToFile(file io.Writer) error {
	if _, err := file.Write(p.Address[:]); err != nil {
		return errors.Wrap(err, "failed to write address")
	}

	buf := make([]byte, 4)
	buf[0] = byte(p.M)
	if _, err := file.Write(buf[:1]); err != nil {
		return errors.Wrap(err, "failed to write chunk size M")
	}

	binary.LittleEndian.PutUint16(buf[:2], p.N)
	if _, err := file.Write(buf[:2]); err != nil {
		return errors.Wrap(err, "failed to write row size N")
	}

	buf[0] = byte(p.K)
	if _, err := file.Write(buf[:1]); err != nil {
		return errors.Wrap(err, "failed to write iteration count K")
	}

	binary.LittleEndian.PutUint32(buf, p.D)
	if _, err := file.Write(buf); err != nil {
		return errors.Wrap(err, "failed to write difficulty")
	}

	return nil
}

func (p *Parameters) Check() bool {
	return p.M == M && p.N == N && p.K == K && p.D <= D
}
