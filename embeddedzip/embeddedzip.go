// embeddedzip provides a convenient way to read a zip embedded in a go
// executable by appending the zip file to the end of the executable file.
// Ordinarily this can be accomplished by simply opening the current executable
// file with archive/zip; unfortunately, at least as of go 1.8, archive/zip is
// too strict about the layout of the zip file and will refuse to read such
// embedded zip files. This package works by locating the start of the embedded
// zip file, and providing a filtered view containing only the zip data to
// archive/zip.
package embeddedzip

import (
	"archive/zip"
	"encoding/binary"
	"errors"
	"io"
	"os"
)

var (
	ErrNoFooter = errors.New("cannot find zip footer; file does not have embedded zip or zip file has comment")
)

const zipFooterSignature = 0x06054b50

type zipFooter struct {
	Signature            uint32
	DiskNum              uint16
	DiskStart            uint16
	NumRecords           uint16
	TotalRecords         uint16
	DirectorySize        uint32
	DirectoryStartOffset uint32
	CommentLength        uint16
}

var zipFooterSize = binary.Size(zipFooter{})

func (f *zipFooter) Verify() error {
	if f.Signature != zipFooterSignature || f.CommentLength != 0 {
		return ErrNoFooter
	}
	return nil
}

func (f *zipFooter) FileSize() int64 {
	return int64(uint32(zipFooterSize) + f.DirectorySize + f.DirectoryStartOffset)
}

func (f *zipFooter) CalculateStartOffset(totalLength int64) (int64, error) {
	zipSize := f.FileSize()
	if zipSize > totalLength {
		return 0, ErrNoFooter
	}
	return totalLength - zipSize, nil
}

type ZipReaderCloser struct {
	*zip.Reader
	io.Closer
}

// OpenEmbeddedZip opens and returns the zip file embedded in the current go
// executable. If no such embedded zip file exists, ErrNoFooter should be
// returned.
func OpenEmbeddedZip() (*ZipReaderCloser, error) {
	f, err := os.Open(os.Args[0])
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			f.Close()
		}
	}()
	length, err := f.Seek(0, 2)
	if err != nil {
		return nil, err
	}
	if length < int64(zipFooterSize) {
		return nil, ErrNoFooter
	}

	_, err = f.Seek(-int64(zipFooterSize), 1)
	if err != nil {
		return nil, err
	}

	var footer zipFooter
	err = binary.Read(f, binary.LittleEndian, &footer)
	if err != nil {
		return nil, err
	}
	err = footer.Verify()
	if err != nil {
		return nil, err
	}

	startOffset, err := footer.CalculateStartOffset(length)
	if err != nil {
		return nil, err
	}

	reader := io.NewSectionReader(f, startOffset, footer.FileSize())
	zipReader, err := zip.NewReader(reader, footer.FileSize())
	return &ZipReaderCloser{zipReader, f}, nil
}
