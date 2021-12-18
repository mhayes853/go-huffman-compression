package compress

import (
	"errors"
	"fmt"
	"io"
	"os"

	// "sync"

	"io.whypeople/huffman/common"
)

// Return type for compressed file
type HuffCompressedFile struct {
	file *os.File
	header *common.HuffHeader
	err error
}

// CompressFile returns a data type with information about the compressed file
// infile: The file to be compressed
// outfile: The file to write the compressed data to
func CompressFile(infile *os.File, outfile *os.File, maxGoroutines int) HuffCompressedFile {
	compressedFile := HuffCompressedFile{outfile, nil, nil}

	// Make sure file pointers are valid
	if infile == nil || outfile == nil {
		compressedFile.err = errors.New("infile and outfile cannot be nil")
		return compressedFile
	}

	// Build Histogram
	histogram, err := buildHistogramConcurrentlyFromFile(infile, maxGoroutines)
	if err != nil {
		compressedFile.err = err
		return compressedFile
	}
	
	for i := 0; i < common.ALPHABET_SIZE; i++ {
		w := histogram[byte(i)]
		if w > 0 {
			fmt.Println(string(rune(i)), w)
		}
	}

	return compressedFile
}

// buildHistogramConcurrentlyFromFile builds a histogram from a file using the specified number of maxGoroutines
// infile: The file to build the histogram from
// maxGoroutines: The number of goroutines to use to build the histogram concurrently
func buildHistogramConcurrentlyFromFile(infile *os.File, maxGoroutines int) (map[byte]int, error) {
	histogramChan := make(chan map[byte]int)
	errChan := make(chan error)

	// Each goroutine will read from the file and build a local histogram
	for i := 0; i < maxGoroutines; i++ {
		go func() {	
			buf := make([]byte, common.READ_BLOCK_SIZE)
			localHist := make(map[byte]int)
			for {
				n, err := infile.Read(buf)
				if err != nil {
					if err == io.EOF {
						// Done reading file, send local histogram down the channel
						histogramChan <- localHist
						break
					} else {
						// Some IO Error, send it down the error channel
						errChan <- err
						return
					}
				}
				// Build local histogram
				for _, b := range buf[:n] {
					localHist[b]++
				}
			}
		}()
	}

	select {
	case err := <-errChan:
		// Handle Error
		return nil, err
	default:
		// Recieve all the local histograms from the goroutines and compile them into 1 histogram
		histogram := make(map[byte]int)
		for i := 0; i < maxGoroutines; i++ {
			hist := <-histogramChan
			for k, v := range hist {
				histogram[k] += v
			}
		}
		return histogram, nil
	}
}