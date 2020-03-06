package core

import (
	"bytes"
	"context"
	"hash/crc32"
	"io/ioutil"

	"github.com/oneconcern/datamon/pkg/storage"
)

// metaObject knows how to write and retrieve metadata from store
type metaObject struct {
	meta      storage.Store
	contexter func() context.Context // contexter produces context.Context as needed, thus avoiding to always pass context
}

func defaultMetaObject(meta storage.Store) metaObject {
	return metaObject{
		meta:      meta,
		contexter: backgroundContexter,
	}
}

// MetaStore yields the metadata store for this index
func (m *metaObject) MetaStore() storage.Store {
	return m.meta
}

// readMetadata retrieves a metaObject from some metadata store
func (m *metaObject) readMetadata(pth string) ([]byte, error) {
	rdr, err := m.meta.Get(m.contexter(), pth)
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(rdr)
}

// writeMetadata puts a metaObject on some metadata store
func (m *metaObject) writeMetadata(pth string, noOverwrite bool, buffer []byte) error {
	switch msCRC := m.meta.(type) {
	case storage.StoreCRC:
		crc := crc32.Checksum(buffer, crc32.MakeTable(crc32.Castagnoli))
		return msCRC.PutCRC(m.contexter(), pth, bytes.NewReader(buffer), noOverwrite, crc)
	default:
		return msCRC.Put(m.contexter(), pth, bytes.NewReader(buffer), noOverwrite)
	}
}

// backgroundContexter is the default contexter for metaObject
func backgroundContexter() context.Context {
	return context.Background()
}
