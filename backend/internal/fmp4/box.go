// Package fmp4 splits a fragmented MP4 byte stream into an initialization
// segment and media fragments that can be fed to a browser's MediaSource.
package fmp4

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	headerSize      = 8
	largeHeaderSize = 16

	// maxBoxSize guards against a corrupt size field causing a huge allocation.
	maxBoxSize = 64 << 20
)

var ErrBoxTooLarge = errors.New("fmp4: box exceeds size limit")

// Box is a complete top-level ISO BMFF box, header included.
type Box struct {
	Type string
	Data []byte
}

// ReadBox reads the next top-level box from r.
func ReadBox(r io.Reader) (Box, error) {
	var hdr [largeHeaderSize]byte
	if _, err := io.ReadFull(r, hdr[:headerSize]); err != nil {
		return Box{}, err
	}

	size := uint64(binary.BigEndian.Uint32(hdr[0:4]))
	typ := string(hdr[4:8])
	hdrLen := headerSize

	switch size {
	case 0:
		return Box{}, fmt.Errorf("fmp4: open-ended %q box is not supported", typ)
	case 1:
		if _, err := io.ReadFull(r, hdr[headerSize:largeHeaderSize]); err != nil {
			return Box{}, unexpected(err)
		}
		size = binary.BigEndian.Uint64(hdr[8:16])
		hdrLen = largeHeaderSize
	}

	if size < uint64(hdrLen) {
		return Box{}, fmt.Errorf("fmp4: invalid size %d for %q box", size, typ)
	}
	if size > maxBoxSize {
		return Box{}, ErrBoxTooLarge
	}

	data := make([]byte, size)
	copy(data, hdr[:hdrLen])
	if _, err := io.ReadFull(r, data[hdrLen:]); err != nil {
		return Box{}, unexpected(err)
	}
	return Box{Type: typ, Data: data}, nil
}

// findChild returns the payload of the first direct child of the given type
// inside a container payload.
func findChild(payload []byte, typ string) ([]byte, bool) {
	for len(payload) >= headerSize {
		size := int(binary.BigEndian.Uint32(payload[0:4]))
		hdrLen := headerSize
		if size == 1 {
			if len(payload) < largeHeaderSize {
				return nil, false
			}
			size = int(binary.BigEndian.Uint64(payload[8:16]))
			hdrLen = largeHeaderSize
		}
		if size < hdrLen || size > len(payload) {
			return nil, false
		}
		if string(payload[4:8]) == typ {
			return payload[hdrLen:size], true
		}
		payload = payload[size:]
	}
	return nil, false
}

func unexpected(err error) error {
	if errors.Is(err, io.EOF) {
		return io.ErrUnexpectedEOF
	}
	return err
}
