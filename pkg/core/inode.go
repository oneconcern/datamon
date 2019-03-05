package core

import (
	"os"
	"sync"

	"github.com/jacobsa/fuse/fuseops"
)

type iNodeGenerator struct {
	lock         sync.Mutex
	highestInode fuseops.InodeID
	// TODO: Replace with something more memory and cpu efficient
	freeInodes []fuseops.InodeID
}

type lookupEntry struct {
	iNode fuseops.InodeID
	mode  os.FileMode
}

type nodeEntry struct {
	lock              sync.Mutex // TODO: Replace with key based locking.
	refCount          int
	attr              fuseops.InodeAttributes
	pathToBackingFile string // empty for directory
}

func (g *iNodeGenerator) allocINode() fuseops.InodeID {
	g.lock.Lock()
	var n fuseops.InodeID
	if len(g.freeInodes) == 0 {
		g.highestInode++
		n = g.highestInode
	} else {
		n = g.freeInodes[len(g.freeInodes)-1]
		g.freeInodes = g.freeInodes[:len(g.freeInodes)-1]
		if len(g.freeInodes) == 0 {
			g.highestInode = firstINode
		}
	}
	g.lock.Unlock()
	return n
}

func (g *iNodeGenerator) freeINode(iNode fuseops.InodeID) {
	i := iNode
	g.lock.Lock()
	if g.highestInode == i {
		g.highestInode--
	} else {
		g.freeInodes = append(g.freeInodes, i)
	}
	g.lock.Unlock()
}
