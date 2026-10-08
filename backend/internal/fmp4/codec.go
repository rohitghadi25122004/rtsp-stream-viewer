package fmp4

import (
	"errors"
	"fmt"
)

// visualSampleEntrySize is the fixed part of a VisualSampleEntry that
// precedes its child boxes (ISO/IEC 14496-12, 12.1.3).
const visualSampleEntrySize = 78

var ErrUnsupportedCodec = errors.New("fmp4: unsupported codec")

// CodecString returns the RFC 6381 codec parameter (for example
// "avc1.64001f") of the first video track described by a moov box.
func CodecString(moov []byte) (string, error) {
	if len(moov) < headerSize || string(moov[4:8]) != "moov" {
		return "", errors.New("fmp4: not a moov box")
	}

	payload := moov[headerSize:]
	for _, typ := range []string{"trak", "mdia", "minf", "stbl", "stsd"} {
		child, ok := findChild(payload, typ)
		if !ok {
			return "", fmt.Errorf("fmp4: moov has no %s box", typ)
		}
		payload = child
	}

	// stsd is a full box: version/flags followed by an entry count.
	if len(payload) < 8+headerSize {
		return "", errors.New("fmp4: empty stsd box")
	}
	entries := payload[8:]
	format := string(entries[4:8])

	switch format {
	case "avc1", "avc3":
		if len(entries) < headerSize+visualSampleEntrySize {
			return "", errors.New("fmp4: truncated avc sample entry")
		}
		avcC, ok := findChild(entries[headerSize+visualSampleEntrySize:], "avcC")
		if !ok || len(avcC) < 4 {
			return "", errors.New("fmp4: missing avcC box")
		}
		// avcC: configurationVersion, profile, compatibility, level.
		return fmt.Sprintf("%s.%02x%02x%02x", format, avcC[1], avcC[2], avcC[3]), nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedCodec, format)
	}
}
