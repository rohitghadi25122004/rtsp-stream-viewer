package fmp4

import (
	"bufio"
	"errors"
	"io"
)

// Init is the initialization segment (ftyp + moov) of a stream.
type Init struct {
	Data  []byte
	Codec string
}

// Segmenter groups the boxes of a fragmented MP4 stream into an
// initialization segment followed by media fragments.
type Segmenter struct {
	r *bufio.Reader
}

func NewSegmenter(r io.Reader) *Segmenter {
	return &Segmenter{r: bufio.NewReaderSize(r, 256<<10)}
}

// ReadInit consumes boxes up to and including moov. It must be called once,
// before ReadFragment.
func (s *Segmenter) ReadInit() (Init, error) {
	var data []byte
	for {
		box, err := ReadBox(s.r)
		if err != nil {
			return Init{}, err
		}
		switch box.Type {
		case "ftyp":
			data = append(data, box.Data...)
		case "moov":
			if data == nil {
				return Init{}, errors.New("fmp4: moov before ftyp")
			}
			codec, err := CodecString(box.Data)
			if err != nil {
				return Init{}, err
			}
			return Init{Data: append(data, box.Data...), Codec: codec}, nil
		}
	}
}

// ReadFragment returns the next moof+mdat pair. Other top-level boxes are
// skipped.
func (s *Segmenter) ReadFragment() ([]byte, error) {
	var moof []byte
	for {
		box, err := ReadBox(s.r)
		if err != nil {
			return nil, err
		}
		switch box.Type {
		case "moof":
			moof = box.Data
		case "mdat":
			if moof == nil {
				continue
			}
			return append(moof, box.Data...), nil
		}
	}
}
