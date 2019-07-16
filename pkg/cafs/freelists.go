package cafs

import (
	"sync"
)

/* this is a memory pooling technique adapted from
 * github.com/jacobsa/fuse/internal/freelist.
 */

/* the leaf freelist uses a fixed size (via upper bound on cafs leaf size) buffer
 * for simplicity:   under the assumption is that there are relatively few
 * leaves in flight at a given time, increase `MaxLeafSize` to maintain.
 */

/* use the slice to access the backing array */
type leafBuffer struct {
	buf   [MaxLeafSize]byte
	slice []byte
}

/* this is the pool of memory:  adding buffers to the list means that
 * they can be re-used.
 */
type leafFreelist struct {
	list []*leafBuffer
	mu   sync.Mutex
}

func newLeafFreelist() *leafFreelist {
	return &leafFreelist{
		list: make([]*leafBuffer, 0),
	}
}

/* allocate memory */
func (l *leafFreelist) get() *leafBuffer {
	var x *leafBuffer
	if l != nil {
		l.mu.Lock()
		ll := len(l.list)
		if ll != 0 {
			x = l.list[ll-1]
			l.list = l.list[:ll-1]
		}
		l.mu.Unlock()
	}
	if x == nil {
		x = new(leafBuffer)
	}
	x.slice = x.buf[:]
	x.slice = x.slice[:0]
	return x
}

/* return memory to pool */
func (l *leafFreelist) put(lb *leafBuffer) {
	if l != nil {
		l.mu.Lock()
		l.list = append(l.list, lb)
		l.mu.Unlock()
	}
}
