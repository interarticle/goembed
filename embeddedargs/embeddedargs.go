// Package embeddedargs allows you to set the default command-line args (flags)
// for a go binary by putting a arguments.txt in an embedded zip file
// compatible with embeddedzip.
// The arguments file must be at the base of the embedded zip file, with the
// name "arguments.txt". This file is read as space-delimited CSV, with each
// field being a single argument. Multiple lines in the file are joined
// together.
// Arguments in the file are prepended to os.Args (after argv[0]).
package embeddedargs

import (
	"encoding/csv"
	"io"
	"os"

	"github.com/interarticle/goembed/embeddedzip"
)

const (
	kArgumentsFileName = "arguments.txt"
)

// LoadEmbeddedArguments loads arguments from the embedded zip file into
// os.Args.
// No error is returned if the executable file does not have an embedded zip
// file.
func LoadEmbeddedArguments() error {
	zf, err := embeddedzip.OpenEmbeddedZip()
	if err != nil {
		if err == embeddedzip.ErrNoFooter {
			return nil
		}
		return err
	}
	defer zf.Close()

	for _, f := range zf.File {
		if f.Name == kArgumentsFileName {
			r, err := f.Open()
			if err != nil {
				return err
			}
			defer r.Close()
			csvr := csv.NewReader(r)
			csvr.Comma = ' '

			var arguments []string
			for {
				rec, err := csvr.Read()
				if err != nil {
					if err == io.EOF {
						break
					}
					return err
				}
				arguments = append(arguments, rec...)
			}

			os.Args = append(append([]string{os.Args[0]}, arguments...), os.Args[1:]...)
			return nil
		}
	}
	return nil
}
