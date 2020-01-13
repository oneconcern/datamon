package core

import (
	"os"
	"sync"
	"testing"
	"time"

	iradix "github.com/hashicorp/go-immutable-radix"
	"github.com/spf13/afero"

	"github.com/stretchr/testify/assert"

	"github.com/jacobsa/fuse/fuseops"

	"github.com/jacobsa/fuse/fuseutil"

	"github.com/stretchr/testify/require"
)

type LookupKeys struct {
	iNode    fuseops.InodeID
	name     string
	expected []byte
}

var lookupKeys = []LookupKeys{
	// Already clean
	{0, "0", []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x30}},
	{1, "0", []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x30}},
	{16, "2", []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x32}},
	{18446744073709551615, "key", []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x6b, 0x65, 0x79}},
}

func TestFormLookupKey(t *testing.T) {
	for _, test := range lookupKeys {
		keyGenerated := formLookupKey(test.iNode, test.name)
		require.Equal(t, test.expected, keyGenerated)
	}
}

func TestCreate(t *testing.T) {
	const testRoot = "../../testdata/core"
	child := "child"
	fs := fsMutable{
		fsCommon: fsCommon{
			bundle:     nil,
			lookupTree: iradix.New(),
		},
		iNodeStore: iradix.New(),
		readDirMap: make(map[fuseops.InodeID]map[fuseops.InodeID]*fuseutil.Dirent),
		lock:       sync.Mutex{},
		iNodeGenerator: iNodeGenerator{
			lock:         sync.Mutex{},
			highestInode: firstINode,
			freeInodes:   make([]fuseops.InodeID, 0, 1024), // equates to 1024 files deleted
		},
		localCache:   afero.NewBasePathFs(afero.NewOsFs(), testRoot+"/fs"),
		backingFiles: make(map[fuseops.InodeID]*afero.File),
	}
	fs.iNodeStore, _, _ = fs.iNodeStore.Insert(formKey(firstINode), &nodeEntry{
		attr: fuseops.InodeAttributes{
			Size:   64,
			Nlink:  dirLinkCount,
			Mode:   dirDefaultMode,
			Atime:  time.Time{},
			Mtime:  time.Time{},
			Ctime:  time.Time{},
			Crtime: time.Time{},
			Uid:    defaultUID,
			Gid:    defaultGID,
		},
	})
	childInodeEntry := fuseops.ChildInodeEntry{}
	parent := firstINode
	err := fs.createNode(nil, parent, child, &childInodeEntry, fuseutil.DT_Directory, false)
	assert.NoError(t, err)
	validateChild(t, child, &fs, parent, 3, firstINode+1, 1, &childInodeEntry)

	child2 := "child2"
	err = fs.createNode(nil, parent, child2, &childInodeEntry, fuseutil.DT_Directory, false)
	assert.NoError(t, err)

	validateChild(t, child2, &fs, parent, 4, firstINode+2, 2, &childInodeEntry)

	child3 := "child3"
	err = fs.createNode(nil, parent, child3, &childInodeEntry, fuseutil.DT_Directory, false)
	assert.NoError(t, err)
	validateChild(t, child3, &fs, parent, 5, firstINode+3, 3, &childInodeEntry)
}

func validateChild(t *testing.T, name string, fs *fsMutable, parent fuseops.InodeID, linkCount uint32, childID fuseops.InodeID, childCount int, entry *fuseops.ChildInodeEntry) {
	// Test parent link count increment
	p, _ := fs.iNodeStore.Get(formKey(parent))
	assert.NotNil(t, p)
	parentEntry := p.(*nodeEntry)
	assert.NotNil(t, parentEntry)
	assert.Equal(t, linkCount, parentEntry.attr.Nlink)

	// Test child node entry
	c, _ := fs.iNodeStore.Get(formKey(childID))
	assert.NotNil(t, c)
	childEntry := c.(*nodeEntry)

	var lc uint32 = 1
	var mode os.FileMode = fileDefaultMode
	var s uint64
	var nodeType = fuseutil.DT_File
	if entry.Attributes.Mode.IsDir() {
		lc = dirLinkCount
		mode = dirDefaultMode
		s = dirInitialSize
		nodeType = fuseutil.DT_Directory
	}
	expectedChildEntry := nodeEntry{
		attr: fuseops.InodeAttributes{
			Size:   s,
			Nlink:  lc,
			Mode:   mode,
			Atime:  time.Time{},
			Mtime:  time.Time{},
			Ctime:  time.Time{},
			Crtime: time.Time{},
			Uid:    defaultUID,
			Gid:    defaultGID,
		},
	}
	assert.Equal(t, expectedChildEntry.attr.Nlink, childEntry.attr.Nlink)
	assert.Equal(t, expectedChildEntry.attr.Mode, childEntry.attr.Mode)
	assert.Equal(t, expectedChildEntry.attr.Size, childEntry.attr.Size)
	assert.Equal(t, expectedChildEntry.attr.Gid, childEntry.attr.Gid)
	assert.Equal(t, expectedChildEntry.attr.Uid, childEntry.attr.Uid)

	// Test lookup
	l, _ := fs.lookupTree.Get(formLookupKey(parent, name))
	assert.Equal(t, childID, l.(lookupEntry).iNode)

	// Test ReadDir
	exDE := fuseutil.Dirent{
		Offset: fuseops.DirOffset(childCount),
		Inode:  childID,
		Name:   name,
		Type:   nodeType,
	}
	child := fs.readDirMap[firstINode][childID]
	assert.Equal(t, childCount, len(fs.readDirMap[firstINode]))
	assert.Equal(t, exDE.Inode, child.Inode)
	assert.Equal(t, exDE.Name, child.Name)
	assert.Equal(t, exDE.Type, child.Type)
	assert.Equal(t, expectedChildEntry.attr.Nlink, entry.Attributes.Nlink)
	assert.Equal(t, expectedChildEntry.attr.Mode, entry.Attributes.Mode)
	assert.Equal(t, expectedChildEntry.attr.Size, entry.Attributes.Size)
	assert.Equal(t, expectedChildEntry.attr.Gid, entry.Attributes.Gid)
	assert.Equal(t, expectedChildEntry.attr.Uid, entry.Attributes.Uid)

	// TODO: Add timestamp checks
}
