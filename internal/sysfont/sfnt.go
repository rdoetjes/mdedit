package sysfont

import (
	"encoding/binary"
	"slices"

	ot "github.com/go-text/typesetting/font/opentype"
)

// standalone writes a face as a TrueType file of its own. A collection
// shares tables between its faces, and fpdf can't read collections, so the
// tables are copied into a new file: a directory of tags, checksums and
// positions, followed by the tables.
func standalone(ld *ot.Loader) ([]byte, error) {
	tags := ld.Tables()
	slices.SortFunc(tags, compareTags)
	offset := 12 + 16*len(tags)
	head := make([]byte, offset)
	writeHeader(head, len(tags))
	var body []byte
	for i, tag := range tags {
		t, err := ld.RawTable(tag)
		if err != nil {
			return nil, err
		}
		writeEntry(head[12+16*i:], uint32(tag), t, uint32(offset+len(body)))
		body = append(append(body, t...), make([]byte, -len(t)&3)...)
	}
	return append(head, body...), nil
}

func compareTags(a, b ot.Tag) int { return int(int64(a) - int64(b)) }

func writeHeader(b []byte, n int) {
	shift := 0
	for 1<<(shift+1) <= n {
		shift++
	}
	binary.BigEndian.PutUint32(b, 0x00010000)
	binary.BigEndian.PutUint16(b[4:], uint16(n))
	binary.BigEndian.PutUint16(b[6:], uint16(16<<shift))
	binary.BigEndian.PutUint16(b[8:], uint16(shift))
	binary.BigEndian.PutUint16(b[10:], uint16(n*16-16<<shift))
}

func writeEntry(b []byte, tag uint32, table []byte, offset uint32) {
	binary.BigEndian.PutUint32(b, tag)
	binary.BigEndian.PutUint32(b[4:], checksum(table))
	binary.BigEndian.PutUint32(b[8:], offset)
	binary.BigEndian.PutUint32(b[12:], uint32(len(table)))
}

func checksum(t []byte) uint32 {
	var sum uint32
	for len(t) >= 4 {
		sum += binary.BigEndian.Uint32(t)
		t = t[4:]
	}
	var last [4]byte
	copy(last[:], t)
	return sum + binary.BigEndian.Uint32(last[:])
}
