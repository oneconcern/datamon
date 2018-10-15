package cafs

import (
	"context"
	"io"

	"github.com/oneconcern/datamon/pkg/blob"
)

func newReader(blobs blob.Store, hash Key, leafSize uint32) (io.ReadCloser, error) {
	keys, err := LeafsForHash(blobs, hash, leafSize)
	if err != nil {
		return nil, err
	}
	return &chunkReader{
		fs:       blobs,
		hash:     hash,
		keys:     keys,
		leafSize: leafSize,
	}, nil
}

type chunkReader struct {
	fs       blob.Store
	leafSize uint32

	hash Key

	keys []Key
	idx  int

	rdr       io.ReadCloser
	readSoFar int
	lastChunk bool
}

func (r *chunkReader) Close() error {
	if r.rdr != nil {
		return r.rdr.Close()
	}
	return nil
}

func (r *chunkReader) Read(data []byte) (int, error) {
	bytesToRead := len(data)

	if r.lastChunk && r.rdr == nil {
		return 0, io.EOF
	}
	for {
		key := r.keys[r.idx]
		if r.rdr == nil {
			rdr, err := r.fs.Get(context.Background(), key.String())
			if err != nil {
				return r.readSoFar, err
			}
			r.rdr = rdr
		}

		n, err := r.rdr.Read(data[r.readSoFar:])
		if err != nil {
			//#nosec
			r.rdr.Close()
			r.readSoFar += n
			if err == io.EOF { // we reached the end of the stream for this key
				r.idx++
				r.rdr = nil
				r.lastChunk = r.idx == len(r.keys)

				if r.lastChunk { // this was the last chunk, so also EOF for this hash
					if n == bytesToRead {
						return n, nil
					}
					return r.readSoFar, io.EOF
				}

				// move on to the next key
				continue
			}
			return n, err
		}
		// we filled up the entire byte slice but still have data remaining in the reader,
		// we should move on to receive the next buffer
		r.readSoFar += n
		if r.readSoFar >= bytesToRead {
			r.readSoFar = 0
			// return without error
			return bytesToRead, nil
		}
	}
}
