package fingerprint

import (
	"bytes"
	"io"
	"log"
	"os"
	"runtime"
	"sync"

	units "github.com/docker/go-units"
	blake2b "github.com/minio/blake2b-simd"
)

type chunkInput struct {
	part       int
	partBuffer []byte
	lastChunk  bool
	leafSize   uint32
	level      int
}

type chunkOutput struct {
	digest []byte
	part   int
}

type Option func(*Maker)

func LeafSize(sz int64) Option {
	return func(m *Maker) {
		m.leafSize = uint32(sz)
	}
}

func NumberOfWorkers(no int) Option {
	return func(m *Maker) {
		m.numberOfWorkers = no
	}
}

func Size(sz uint8) Option {
	return func(m *Maker) {
		m.size = sz
	}
}

func New(opts ...Option) *Maker {
	m := &Maker{
		leafSize:        uint32(5 * units.MB),
		numberOfWorkers: runtime.NumCPU(),
		size:            64,
	}

	for _, apply := range opts {
		apply(m)
	}
	return m
}

type Maker struct {
	size            uint8
	leafSize        uint32
	numberOfWorkers int
}

func (m *Maker) Process(path string) (digest []byte, err error) {
	r, size, err := m.openPath(path)
	if err != nil {
		return nil, err
	}
	var wg sync.WaitGroup
	chunks := make(chan chunkInput)
	results := make(chan chunkOutput)

	for i := 0; i < m.numberOfWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.processChunk(chunks, results)
		}()
	}

	go func() {
		for part, totalSize := 0, int64(0); ; part++ {
			partBuffer := make([]byte, m.leafSize)
			n, e := r.Read(partBuffer)
			if e != nil {
				if e == io.EOF {
					break
				}
				return
			}
			partBuffer = partBuffer[:n]

			totalSize += int64(n)
			lastChunk := uint32(n) < m.leafSize || uint32(n) == m.leafSize && totalSize == size

			chunks <- chunkInput{part: part, partBuffer: partBuffer, lastChunk: lastChunk, leafSize: m.leafSize, level: 0}

			if lastChunk {
				break
			}
		}

		// Close input channel
		close(chunks)
	}()

	// Wait for workers to complete
	go func() {
		wg.Wait()
		close(results) // Close output channel
	}()

	// Create hash based on chunk number with digest of chunk
	// (number of chunks upfront is unknown for stdin stream)
	digestHash := make(map[int][]byte)
	for r := range results {
		digestHash[r.part] = r.digest
	}

	// Concatenate digests of chunks
	sz := int(m.size)
	b := make([]byte, len(digestHash)*sz)
	for index, val := range digestHash {
		offset := sz * index
		copy(b[offset:offset+sz], val)
	}

	rootBlake, err := blake2b.New(&blake2b.Config{
		Size: blake2b.Size,
		Tree: &blake2b.Tree{
			Fanout:        0,
			MaxDepth:      2,
			LeafSize:      m.leafSize,
			NodeOffset:    0,
			NodeDepth:     1,
			InnerHashSize: m.size,
			IsLastNode:    true,
		},
	})
	if err != nil {
		return nil, err
	}

	// Compute top level digest
	rootBlake.Reset()
	_, err = io.Copy(rootBlake, bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}
	digest = rootBlake.Sum(nil)

	return digest, nil
}

func (m *Maker) openPath(path string) (io.ReadCloser, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}

	fi, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}

	return f, fi.Size(), nil
}

// Worker routine for computing hash for a chunk
func (m *Maker) processChunk(rx <-chan chunkInput, tx chan<- chunkOutput) {
	for c := range rx {
		blake, err := blake2b.New(&blake2b.Config{
			Size: blake2b.Size,
			Tree: &blake2b.Tree{
				Fanout:        0,
				MaxDepth:      2,
				LeafSize:      c.leafSize,
				NodeOffset:    uint64(c.part),
				NodeDepth:     0,
				InnerHashSize: m.size,
				IsLastNode:    c.lastChunk,
			},
		})
		if err != nil {
			log.Println("Failing to create algorithm: ", err)
			return
		}

		blake.Reset()
		_, err = io.Copy(blake, bytes.NewBuffer(c.partBuffer))
		if err != nil {
			log.Println("Failing to compute hash: ", err)
			tx <- chunkOutput{digest: []byte(""), part: c.part}
		} else {
			digest := blake.Sum(nil)
			tx <- chunkOutput{digest: digest, part: c.part}
		}
	}
}
