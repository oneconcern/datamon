package cafs

import (
	"fmt"
	"sync"
	"testing"
	"unsafe"

	"github.com/docker/go-units"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testWaitRelease(l FreeList) {
	list := l.(*leafFreelist)
	list.wg.Wait()
}

func TestBytesToBuffers(t *testing.T) {
	assert.Equal(t, 5, BytesToBuffers(10000, 2400))
}

func TestFreeList(t *testing.T) {
	xsize := uint32(unsafe.Sizeof(leafBufferMax{}))
	l := newLeafFreelist(MaxLeafSize, 100)
	assert.EqualValues(t, xsize, l.Size())

	buffers := make([]LeafBuffer, 0, 5)

	buffers = append(buffers, l.Get())
	require.Len(t, buffers[0].Bytes(), 0)
	buffers = append(buffers, l.Get())
	require.Len(t, buffers[1].Bytes(), 0)
	buffers = append(buffers, l.Get())
	require.Len(t, buffers[2].Bytes(), 0)
	buffers[0].Pin()
	buffers[2].Pin()
	buffers[2].Copy([]byte(`abcd`))
	assert.Equal(t, 3, l.Buffers())

	// async release, based on buffer idleness
	l.Release(buffers[0])
	assert.Equal(t, 0, l.FreeBuffers())
	assert.Equal(t, 3, l.Buffers())

	buffers[0].Unpin()
	testWaitRelease(l)
	assert.Equal(t, 1, l.FreeBuffers())

	buffers = append(buffers, l.Get())
	require.Len(t, buffers[3].Bytes(), 0)
	assert.Equal(t, buffers[0], buffers[3])
	assert.Equal(t, 3, l.Buffers())

	buffers[2].Unpin()
	l.Release(buffers[2])
	testWaitRelease(l)
	assert.Equal(t, 3, l.Buffers())

	buffers = append(buffers, l.Get())
	assert.Equal(t, 3, l.Buffers())
	require.Len(t, buffers[4].Bytes(), 0)
	assert.Equal(t, buffers[2], buffers[4])

	assert.Equal(t, int(MaxLeafSize), cap(buffers[4].Bytes()))

	// when nil freelist
	var empty *leafFreelist
	l = empty
	b := l.Get()
	require.NotNil(t, b)
	assert.Nil(t, l)
	assert.Equal(t, uint32(0), l.Size())
	assert.Equal(t, 0, l.FreeBuffers())
	assert.Equal(t, 0, l.Buffers())
	l.Release(b)

	// allocating smaller bufers
	for _, toPin := range []struct {
		name         string
		leafSize     uint32
		expectedSize uint32
		expectedType interface{}
		internalPtr  func(LeafBuffer) (string, string, []byte)
	}{
		{
			name:         "buffer 1MB",
			leafSize:     1 * units.KiB,
			expectedSize: 1 * units.MiB,
			expectedType: &leafBuffer1MB{},
			internalPtr: func(x LeafBuffer) (addr1 string, addr2 string, slice []byte) {
				internal, ok := x.(*leafBuffer1MB)
				if !ok {
					return
				}
				addr1 = fmt.Sprintf("%p", &internal.buf)
				addr2 = fmt.Sprintf("%p", internal.slice)
				slice = internal.slice
				return
			},
		},
		{
			name:         "buffer 2MB",
			leafSize:     1025 * units.KiB,
			expectedSize: 2 * units.MiB,
			expectedType: &leafBuffer2MB{},
			internalPtr: func(x LeafBuffer) (addr1 string, addr2 string, slice []byte) {
				internal, ok := x.(*leafBuffer2MB)
				if !ok {
					return
				}
				addr1 = fmt.Sprintf("%p", &internal.buf)
				addr2 = fmt.Sprintf("%p", internal.slice)
				slice = internal.slice
				return
			},
		},
		{
			name:         "buffer 3MB",
			leafSize:     3000 * units.KiB,
			expectedSize: 3 * units.MiB,
			expectedType: &leafBuffer3MB{},
			internalPtr: func(x LeafBuffer) (addr1 string, addr2 string, slice []byte) {
				internal, ok := x.(*leafBuffer3MB)
				if !ok {
					return
				}
				addr1 = fmt.Sprintf("%p", &internal.buf)
				addr2 = fmt.Sprintf("%p", internal.slice)
				slice = internal.slice
				return
			},
		},
		{
			name:         "buffer 4MB",
			leafSize:     3500 * units.KiB,
			expectedSize: 4 * units.MiB,
			expectedType: &leafBuffer4MB{},
			internalPtr: func(x LeafBuffer) (addr1 string, addr2 string, slice []byte) {
				internal, ok := x.(*leafBuffer4MB)
				if !ok {
					return
				}
				addr1 = fmt.Sprintf("%p", &internal.buf)
				addr2 = fmt.Sprintf("%p", internal.slice)
				slice = internal.slice
				return
			},
		},
		{
			name:         "buffer 5MB",
			leafSize:     8000 * units.KiB,
			expectedSize: 5 * units.MiB,
			expectedType: &leafBufferMax{},
			internalPtr: func(x LeafBuffer) (addr1 string, addr2 string, slice []byte) {
				internal, ok := x.(*leafBufferMax)
				if !ok {
					return
				}
				addr1 = fmt.Sprintf("%p", &internal.buf)
				addr2 = fmt.Sprintf("%p", internal.slice)
				slice = internal.slice
				return
			},
		},
	} {
		testCase := toPin
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			testFreeBuffers(t, testCase.leafSize, testCase.expectedSize, testCase.expectedType, testCase.internalPtr)
		})
	}
}

func testFreeBuffers(t *testing.T, leafSize, expectedSize uint32, expectedType interface{}, internalPtr func(LeafBuffer) (string, string, []byte)) {
	var xsize uint32
	switch expectedType.(type) {
	case *leafBuffer1MB:
		// nolint: staticcheck
		d := leafBuffer1MB{}
		xsize = uint32(unsafe.Sizeof(d))
	case *leafBuffer2MB:
		// nolint: staticcheck
		d := leafBuffer2MB{}
		xsize = uint32(unsafe.Sizeof(d))
	case *leafBuffer3MB:
		// nolint: staticcheck
		d := leafBuffer3MB{}
		xsize = uint32(unsafe.Sizeof(d))
	case *leafBuffer4MB:
		// nolint: staticcheck
		d := leafBuffer4MB{}
		xsize = uint32(unsafe.Sizeof(d))
	case *leafBufferMax:
		// nolint: staticcheck
		d := leafBufferMax{}
		xsize = uint32(unsafe.Sizeof(d))
	default:
		panic("wrong type expectations for freelist buffers")
	}
	overhead := uint32(unsafe.Sizeof(int(0))) + uint32(unsafe.Sizeof(sync.Mutex{}))
	alignment := 2 * uint32(unsafe.Sizeof(int(0))) // tolerance for alignment
	require.Truef(t, expectedSize <= xsize && expectedSize >= xsize-overhead-alignment,
		"invalid size expectations: expected size %d isn't close to actual size %d, accounting for overhead: %d", expectedSize, xsize, overhead)

	l := newLeafFreelist(leafSize, 100)
	assert.EqualValues(t, xsize, l.Size())
	assert.EqualValues(t, 0, l.FreeBuffers())
	x := l.Get()
	x.Pin()

	require.IsType(t, expectedType, x)
	require.Len(t, x.Bytes(), 0)

	addr1, addr2, internalSlice := internalPtr(x)
	require.NotEmpty(t, addr1)
	require.NotEmpty(t, addr2)
	require.NotNil(t, internalSlice)
	assert.Equal(t, addr1, addr2)

	assert.Len(t, internalSlice, 0)
	assert.Equal(t, int(expectedSize), cap(internalSlice))

	x.Copy([]byte(`abc`))
	assert.EqualValues(t, []byte(`abc`), x.Bytes())

	addr10, addr20, internalSlice := internalPtr(x)
	require.Len(t, internalSlice, 3)
	assert.EqualValues(t, []byte(`abc`), internalSlice)
	assert.Equal(t, int(expectedSize), cap(internalSlice))

	assert.Equal(t, addr1, addr10)
	assert.Equal(t, addr2, addr20)

	assert.Equal(t, 0, l.FreeBuffers())

	ptr3 := unsafe.Pointer(&internalSlice[0])

	x.Slice(1, 3)
	require.Len(t, x.Bytes(), 2)
	assert.EqualValues(t, []byte(`bc`), x.Bytes())

	addr10, addr20, internalSlice = internalPtr(x)
	assert.Len(t, internalSlice, 2)
	assert.EqualValues(t, []byte(`bc`), internalSlice)

	// backing array hasn't moved
	assert.Equal(t, addr1, addr10)

	// slicing did set the address of the first element of slice just 1 byte away
	// (asserting here we don't allocate this at some other new place)
	ptr3 = unsafe.Pointer(uintptr(ptr3) + 1)
	addr3 := fmt.Sprintf("%p", ptr3)
	assert.Equalf(t, addr3, addr20, "address differ: %s != %s", addr20, addr3)

	l.Release(x)
	assert.Equal(t, 0, l.FreeBuffers())
	x.Unpin()
	testWaitRelease(l)
	assert.Equal(t, 1, l.FreeBuffers())

	y := l.Get()
	assert.Equal(t, 0, l.FreeBuffers())
	require.Len(t, x.Bytes(), 0)

	require.IsType(t, expectedType, y)
	addr11, addr21, internalSlice := internalPtr(y)
	assert.Len(t, internalSlice, 0)
	assert.Equal(t, int(expectedSize), cap(internalSlice))

	assert.Equal(t, addr1, addr11)
	assert.Equal(t, addr2, addr21)
}
