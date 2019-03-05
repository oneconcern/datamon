package filetracker

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type ioRange struct {
	offset int64
	len    int64
}
type read struct {
	length   int64
	fromBase bool
}

type test struct {
	request  ioRange
	expected read
}

type writeTest struct {
	write  ioRange
	keyLen int
	tests  []test
}

var writeSeq = []writeTest{
	{ // Write a byte
		write:  ioRange{0, 1},
		keyLen: 2,
		tests: []test{
			{
				request:  ioRange{offset: 0, len: 5},
				expected: read{length: 1, fromBase: false},
			},
			{
				request:  ioRange{offset: 1, len: 5},
				expected: read{length: 5, fromBase: true},
			},
			{
				request:  ioRange{offset: 0, len: 1},
				expected: read{length: 1, fromBase: false},
			},
			{
				request:  ioRange{offset: 1, len: 100},
				expected: read{length: 100, fromBase: true},
			},
		},
	},
	{ //Extend the write
		write:  ioRange{0, 10},
		keyLen: 2,
		tests: []test{
			{
				request:  ioRange{offset: 0, len: 5},
				expected: read{length: 5, fromBase: false},
			},
			{
				request:  ioRange{offset: 0, len: 6},
				expected: read{length: 6, fromBase: false},
			},
			{
				request:  ioRange{offset: 1, len: 1},
				expected: read{length: 1, fromBase: false},
			},
			{
				request:  ioRange{offset: 10, len: 100},
				expected: read{length: 100, fromBase: true},
			},
		},
	},
	{ // Write a new region
		write:  ioRange{13, 10},
		keyLen: 4,
		tests: []test{
			{
				request:  ioRange{offset: 0, len: 5},
				expected: read{length: 5, fromBase: false},
			},
			{
				request:  ioRange{offset: 0, len: 6},
				expected: read{length: 6, fromBase: false},
			},
			{
				request:  ioRange{offset: 1, len: 1},
				expected: read{length: 1, fromBase: false},
			},
			{
				request:  ioRange{offset: 10, len: 100},
				expected: read{length: 3, fromBase: true},
			},
			{
				request:  ioRange{offset: 0, len: 11},
				expected: read{length: 10, fromBase: false},
			},
			{
				request:  ioRange{offset: 10, len: 3},
				expected: read{length: 3, fromBase: true},
			},
			{
				request:  ioRange{offset: 13, len: 100},
				expected: read{length: 10, fromBase: false},
			},
			{
				request:  ioRange{offset: 23, len: 100},
				expected: read{length: 100, fromBase: true},
			},
		},
	},
	{ // Coalesce down to one region
		write:  ioRange{0, 22},
		keyLen: 2,
		tests: []test{
			{
				request:  ioRange{offset: 0, len: 5},
				expected: read{length: 5, fromBase: false},
			},
			{
				request:  ioRange{offset: 0, len: 6},
				expected: read{length: 6, fromBase: false},
			},
			{
				request:  ioRange{offset: 1, len: 1},
				expected: read{length: 1, fromBase: false},
			},
			{
				request:  ioRange{offset: 0, len: 100},
				expected: read{length: 23, fromBase: false},
			},
			{
				request:  ioRange{offset: 0, len: 11},
				expected: read{length: 11, fromBase: false},
			},
			{
				request:  ioRange{offset: 10, len: 3},
				expected: read{length: 3, fromBase: false},
			},
			{
				request:  ioRange{offset: 22, len: 100},
				expected: read{length: 1, fromBase: false},
			},
			{
				request:  ioRange{offset: 23, len: 100},
				expected: read{length: 100, fromBase: true},
			},
		},
	},
}

func TestGetKey(t *testing.T) {
	b := getKey(1)
	assert.Equal(t, b[7], uint8(1))
	assert.Equal(t, b[0], uint8(0))
}

func TestTrackWrite(t *testing.T) {
	tf := newTFile(nil, nil, "file")

	for i, w := range writeSeq {
		tf.trackWrite(w.write.offset, w.write.len)
		assert.Equal(t, w.keyLen, tf.tracker.Len())
		for j, test := range w.tests {
			c, s := tf.getRangeToRead(test.request.offset, test.request.len)
			assert.Equal(t, test.expected.length, c, fmt.Sprintf("Failed write:%d, test:%d", i, j))
			if test.expected.fromBase {
				assert.Equal(t, s, base, fmt.Sprintf("Failed write:%d, test:%d", i, j))
			} else {
				assert.Equal(t, s, mutable, fmt.Sprintf("Failed write:%d, test:%d", i, j))
			}
		}
	}
}
