package cafs

import (
	"math"
	"sync"
	"unsafe"

	"github.com/docker/go-units"
)

const (
	leaf1MB uint32 = 1 * units.MiB
	leaf2MB uint32 = 2 * units.MiB
	leaf3MB uint32 = 3 * units.MiB
	leaf4MB uint32 = 4 * units.MiB
)

// BytesToBuffers returns the number of buffers for a given
// size target in bytes, given a leaf size. Buffers are rounded up,
// so we may exceed the target by at most the size of a leaf.
func BytesToBuffers(target int, leafSize uint32) int {
	return int(math.Ceil(float64(target) / float64(leafSize)))
}

// FreeList knows how to allocate buffers and release them
type FreeList interface {
	// Get some recycled idle buffer from the freelist or allocates a new one.
	// Buffers are automatically set to their busy state.
	Get() LeafBuffer

	// Release a buffer to the freelist. Release is asynchronous and buffers get eventually released only when idled.
	Release(LeafBuffer)

	// Buffers tells how many buffers are currently allocated
	Buffers() int

	// FreeBuffers tells how many free buffers are on the free list
	FreeBuffers() int

	// Size yield the fixed buffer size of this free list
	Size() uint32
}

// LeafBuffer represents a buffer allocated once to the memory pool.
//
// It allocates a fixed array and exposes a slice backed by this array for consuming callers.
//
// Usage:
//  * Bytes() buffer method to access the backing array as a slice
//  * Copy() and Slice() buffer methods to modify the content of the buffer
//  * Pin() to pin the buffer and prevent it from being recycled while in a critical section
//  * Unpin() to mark a buffer recyclable
type LeafBuffer interface {
	Bytes() []byte
	Copy([]byte)
	Slice(low, high int) []byte
	Reset()
	Pin()
	Unpin()
}

type baseBuffer struct {
	slice []byte
	busy  sync.Mutex
}

func (m *baseBuffer) Bytes() []byte {
	return m.slice
}

// Copy a buffer's content into a pre-allocated buffer
func (m *baseBuffer) Copy(val []byte) {
	n := copy(m.slice[:cap(m.slice)], val)
	m.slice = m.slice[:n]
}

// Slice a buffer like a regular slice, e.g. with syntax: slice[low:high]
func (m *baseBuffer) Slice(low, high int) []byte {
	m.slice = m.slice[low:high]
	return m.slice
}

// Pin marks a buffer as busy, so it won't be reused before being released to an idle state
// by the same go routine
func (m *baseBuffer) Pin() {
	if m == nil {
		return
	}
	m.busy.Lock()
}

// Unpin marks a buffer idle, so it can be recycled
func (m *baseBuffer) Unpin() {
	if m == nil {
		return
	}
	m.busy.Unlock()
}

type leafBuffer1MB struct {
	baseBuffer
	buf [leaf1MB]byte
}

// Size is the raw size in bytes for this buffer type
func (lb1 *leafBuffer1MB) Size() uint32 {
	return uint32(unsafe.Sizeof(leafBuffer1MB{}))
}

// Reset a buffer (actual content is not scratched)
func (lb1 *leafBuffer1MB) Reset() {
	lb1.slice = lb1.buf[:0]
}

type leafBuffer2MB struct {
	baseBuffer
	buf [leaf2MB]byte
}

func (lb2 *leafBuffer2MB) Size() uint32 {
	return uint32(unsafe.Sizeof(leafBuffer2MB{}))
}

func (lb2 *leafBuffer2MB) Reset() {
	lb2.slice = lb2.buf[:0]
}

type leafBuffer3MB struct {
	baseBuffer
	buf [leaf3MB]byte
}

func (lb3 *leafBuffer3MB) Size() uint32 {
	return uint32(unsafe.Sizeof(leafBuffer3MB{}))
}

func (lb3 *leafBuffer3MB) Reset() {
	lb3.slice = lb3.buf[:0]
}

type leafBuffer4MB struct {
	baseBuffer
	buf [leaf4MB]byte
}

func (lb4 *leafBuffer4MB) Size() uint32 {
	return uint32(unsafe.Sizeof(leafBuffer4MB{}))
}

func (lb4 *leafBuffer4MB) Reset() {
	lb4.slice = lb4.buf[:0]
}

type leafBufferMax struct {
	baseBuffer
	buf [MaxLeafSize]byte
}

func (lbm *leafBufferMax) Size() uint32 {
	return uint32(unsafe.Sizeof(leafBufferMax{}))
}

func (lbm *leafBufferMax) Reset() {
	lbm.slice = lbm.buf[:0]
}

func allocatorFunc(leafSize uint32) func() LeafBuffer {
	switch {
	case leafSize <= 1*units.MiB:
		return func() LeafBuffer {
			x := new(leafBuffer1MB)
			x.Reset()
			return x
		}
	case leafSize <= 2*units.MiB:
		return func() LeafBuffer {
			x := new(leafBuffer2MB)
			x.Reset()
			return x
		}
	case leafSize <= 3*units.MiB:
		return func() LeafBuffer {
			x := new(leafBuffer3MB)
			x.Reset()
			return x
		}
	case leafSize <= 4*units.MiB:
		return func() LeafBuffer {
			x := new(leafBuffer4MB)
			x.Reset()
			return x
		}
	default:
		return func() LeafBuffer {
			x := new(leafBufferMax)
			// back the slice against the array with full capacity;
			// reset the number of elements in the slice
			x.Reset()
			return x
		}
	}
}

func sizeFunc(leafSize uint32) func() uint32 {
	switch {
	case leafSize <= 1*units.MiB:
		t := &leafBuffer1MB{}
		x := t.Size()
		return func() uint32 {
			return x
		}
	case leafSize <= 2*units.MiB:
		t := &leafBuffer2MB{}
		x := t.Size()
		return func() uint32 {
			return x
		}
	case leafSize <= 3*units.MiB:
		t := &leafBuffer3MB{}
		x := t.Size()
		return func() uint32 {
			return x
		}
	case leafSize <= 4*units.MiB:
		t := &leafBuffer4MB{}
		x := t.Size()
		return func() uint32 {
			return x
		}
	default:
		t := &leafBufferMax{}
		x := t.Size()
		return func() uint32 {
			return x
		}
	}
}

// leafFreelist is a memory pool inspired bygithub.com/jacobsa/fuse/internal/freelist.
//
// This pool allows for minimal allocations to be required from the garbage collector
// when a consuming cache (e.g. LRU) requests new buffers or relinquishes some older ones.
//
// The leaf freelist uses a fixed size buffer, which size is coarsely tuned according to the leaf size:
// buffer comes in 5 different sizes from 1MB to 5MB.
//
// The assumption is that there are relatively few
// leaves in flight at a given time.
//
// The nil instance behaves as a regular new allocator, without buffer recycling.
//
// Usage:
//  * Get() to get a bufer
//  * Release() to mark a buffer for releasing to the free list.
//
// Releasing buffers to the list means that they can be re-used whenever idle.
//
// This implementation does not guarantee a bounded number of allocated buffers: however, whenever
// the  freelist reaches it high-watermark, free buffers are relinquished to gc rather than kept
// allocated. This allows for swallowing bursts of parallel accesses.
type leafFreelist struct {
	list      []LeafBuffer
	allocate  func() LeafBuffer
	size      func() uint32
	mu        sync.Mutex
	buffers   int
	watermark int

	// wg is a WaitGroup to be able to test async releases
	// This is a small overhead to get things testable
	wg sync.WaitGroup
}

var _ FreeList = &leafFreelist{}

func newLeafFreelist(leafSize uint32, watermark int) FreeList {
	return &leafFreelist{
		list:      make([]LeafBuffer, 0, watermark),
		allocate:  allocatorFunc(leafSize),
		size:      sizeFunc(leafSize),
		watermark: watermark,
	}
}

func (l *leafFreelist) Buffers() int {
	if l == nil {
		return 0
	}
	l.mu.Lock()
	s := l.buffers
	l.mu.Unlock()
	return s
}

func (l *leafFreelist) FreeBuffers() int {
	if l == nil {
		return 0
	}
	l.mu.Lock()
	s := len(l.list)
	l.mu.Unlock()
	return s
}

func (l *leafFreelist) Size() uint32 {
	if l == nil {
		return 0
	}
	return l.size()
}

// Get allocates a new buffer from memory, or yields are recycled one.
//
// NOTE: it is the caller's responsibility, when pinning a buffer as busy, to idle the buffer
// so the free list can recycle it.
func (l *leafFreelist) Get() LeafBuffer {
	if l == nil {
		lb := allocatorFunc(MaxLeafSize)()
		lb.Reset()
		return lb
	}

	var lb LeafBuffer
	l.mu.Lock()
	ll := len(l.list)
	if ll != 0 {
		lb = l.list[ll-1]
		lb.Reset()
		l.list = l.list[:ll-1]
	} else {
		l.buffers++
	}
	l.mu.Unlock()
	if lb == nil {
		// allocate a new buffer
		lb = l.allocate()
	}
	return lb
}

// Release returns memory to pool (but not to gc).
//
// Release is asynchronous: it promises to eventually return the buffer to
// the free list when idled.
func (l *leafFreelist) Release(lb LeafBuffer) {
	if l == nil {
		return
	}
	l.wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		lb.Pin()
		l.mu.Lock()
		lb.Unpin()
		if len(l.list) > l.watermark {
			lb = nil
		} else {
			l.list = append(l.list, lb)
		}
		l.mu.Unlock()
	}(&l.wg)
}
