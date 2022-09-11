package cafs

import (
	"context"
	"fmt"
	"hash/crc32"

	"github.com/oneconcern/datamon/pkg/errors"
	"github.com/oneconcern/datamon/pkg/storage"
	"go.uber.org/zap"
)

// existsAndValidBlob verifies a blob chunk against its expected size and CRC32C hash, for stores that support CRC.
func existsAndValidBlob(ctx context.Context, store storage.Store, pth string, data []byte, lg *zap.Logger) (found bool, overwrite bool) {
	attr, err := store.GetAttr(ctx, pth)
	found = err == nil

	switch {
	// edge cases detection: figure out possible corruptions
	// this detects edge cases in which the blob has already been created by some previous upload.

	case found && attr.Size == 0:
		// the previous upload somehow that upload did not end up correctly: the object is there but empty.
		// This is the simplest case, and easiest to detect. It is also the only case that we have been
		// able to actually see over 4 years using cafs.
		lg.Warn("cafs found the same root key for this hash blob, but it's empty! About to overwrite it")

		overwrite = true

	case found && attr.Size > 0 && attr.CRC32C > 0:
		// inspect the CRC32C hash, for backends that support it (e.g. not local fs)
		crc := crc32.Checksum(data, crc32.MakeTable(crc32.Castagnoli))
		if crc != attr.CRC32C {
			lg.Warn("cafs found the same root key for this hash blob, but the content's CRC32 hash doesn't match. About to overwrite it")

			overwrite = true
		}

		// the previous upload somehow that upload did not end up correctly:
		// 1. either we got an incomplete write.
		// 2. or we got a hash collision: this key conflicts with some other data item. This is extremely unlikely
		// unless we hit a bug in the Blake2D hashing function.
		//
		// (1) is unlikely but not impossible. (2) is so much more unlikely than (1) than we may consider it impossible, as compared to (1).
		//
		// NOTE(fred): a few thorough analysis papers study collisions when using Blake2B.
		// https://eprint.iacr.org/2013/467.pdf.
		// https://link.springer.com/content/pdf/10.1007/978-3-642-38980-1_8.pdf
		// Collision probabilities indicated there are however super unlikely, e.g << 2^-480
		// While this analysis is conducted in the context of attacking the hash function (i.e. finding the complexity of an algorithm that finds a collision),
		// these metrics can be roughly used as a upper-bound for the probability of a random collision, which should be actually be much lower.
	}

	return found, overwrite
}

// verifyBlob verifies that a written blob is as expected.
func verifyBlob(ctx context.Context, store storage.Store, pth string, data []byte, lg *zap.Logger) error {
	attr, err := store.GetAttr(ctx, pth)
	if err != nil {
		return fmt.Errorf("could not retrieve blob key (%q): %w", pth, err)
	}

	if attr.Size != int64(len(data)) {
		err = errors.New("verification of blob content failed: sizes differ")

		lg.Error("cafs flush with root key error",
			zap.Error(err),
			zap.Int("written_size", len(data)),
			zap.Int64("read_size", attr.Size),
		)

		return err
	}

	crc := crc32.Checksum(data, crc32.MakeTable(crc32.Castagnoli))
	if attr.CRC32C > 0 && crc != attr.CRC32C {
		err = errors.New("verification of blob content failed: CRC32 differ")

		lg.Error("cafs flush with blob checksum error",
			zap.Error(err),
			zap.Int("expected_size", len(data)),
			zap.Int64("reread_size", attr.Size),
			zap.Uint32("expected_crc32", crc),
			zap.Uint32("reread_crc32", attr.CRC32C),
		)

		return err
	}

	return nil
}
