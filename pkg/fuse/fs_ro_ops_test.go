// +build fsintegration

package fuse

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/oneconcern/datamon/pkg/core/mocks"
	"go.uber.org/zap"
)

func fsROActions(pth string, info os.FileInfo, e chan<- error, wg *sync.WaitGroup) {
	// randomized execution of fs actions (test scenarios below) given a walked file on the mount
	//
	// The scenarios defined here are for a read-only mount (writes are exepcted to fail)
	l := mocks.TestLogger()
	defer wg.Done()
	actions := map[string]func(string, string, bool, chan<- error){
		"stat":               testStat,
		"bad-stat":           testBadStat,
		"readFile":           testReadFile,
		"bad-readFile":       testBadReadFile,
		"bad-overwriteFile":  testBadOverwriteFile,
		"bad-createFile":     testBadCreateFile,
		"bad-createFile2":    testBadCreateFile2,
		"bad-mkdir":          testBadMkdir,
		"bad-truncate":       testBadTruncate,
		"bad-chown":          testBadChown,
		"bad-chmod":          testBadChmod,
		"bad-remove":         testBadRemove,
		"bad-rename":         testBadRename,
		"bad-symlink":        testBadSymlink,
		"open-read-seek":     testOpenReadSeek,
		"bad-open-write":     testBadOpenWrite,
		"bad-open-overwrite": testBadOpenOverwrite,
		"bad-open-create":    testBadOpenCreate,
		"statfs":             testStatFS,
	}
	for action, fn := range actions {
		l.Info("fs-action", zap.String("action", action), zap.String("file", pth))
		fn(action, pth, info.IsDir(), e)
	}
}

func sibling(pth, target string) string {
	return filepath.Join(filepath.Dir(pth), target)
}

func testStat(action, pth string, _ bool, e chan<- error) {
	_, err := os.Stat(pth)
	if err != nil {
		e <- fmt.Errorf("%s:cannot stat: %s: %w", action, pth, err)
	}
}

func testBadStat(action, pth string, _ bool, e chan<- error) {
	_, err := os.Stat(sibling(pth, "nowhere"))
	if err == nil {
		e <- fmt.Errorf("%s:expected error on non existent file stat", action)
	}
}

func testReadFile(action, pth string, isDir bool, e chan<- error) {
	if isDir {
		return
	}
	_, err := ioutil.ReadFile(pth)
	if err != nil {
		e <- fmt.Errorf("%s:cannot read: %s: %w", action, pth, err)
	}
}

func testBadReadFile(action, pth string, _ bool, e chan<- error) {
	_, err := ioutil.ReadFile(sibling(pth, "nowhere"))
	if err == nil {
		e <- fmt.Errorf("%s:expected error on reading non existent file", action)
	}
}

func testBadOverwriteFile(action, pth string, isDir bool, e chan<- error) {
	if isDir {
		return
	}
	err := ioutil.WriteFile(pth, []byte("sample"), 0644)
	if err == nil {
		e <- fmt.Errorf("%s:expected RO mount but could overwrite file: %s", action, pth)
	}
}

func testBadCreateFile(action, pth string, _ bool, e chan<- error) {
	err := ioutil.WriteFile(sibling(pth, "test"), []byte("sample"), 0644)
	if err == nil {
		e <- fmt.Errorf("%s:expected RO mount but could create and write file", action)
	}
}

func testBadCreateFile2(action, pth string, _ bool, e chan<- error) {
	_, err := os.Create(sibling(pth, "created"))
	if err == nil {
		e <- fmt.Errorf("%s:expected RO mount but could create file", action)
	}
}

func testBadMkdir(action, pth string, _ bool, e chan<- error) {
	err := os.MkdirAll(sibling(pth, "test-dir"), 0550)
	if err == nil {
		e <- fmt.Errorf("%s:expected RO mount but could create dir", action)
	}
}

func testBadTruncate(action, pth string, isDir bool, e chan<- error) {
	if isDir {
		return
	}
	err := os.Truncate(pth, 0)
	if err == nil {
		e <- fmt.Errorf("%s:expected RO mount but could truncate file: %s", action, pth)
		return
	}
}

func testBadChown(action, pth string, _ bool, e chan<- error) {
	err := os.Chown(pth, 0, 0)
	if err == nil {
		e <- fmt.Errorf("%s:expected RO mount but could chown file: %s", action, pth)
	}
}

func testBadChmod(action, pth string, _ bool, e chan<- error) {
	err := os.Chmod(pth, 0755)
	if err == nil {
		e <- fmt.Errorf("%s:expected RO mount but could chmod file: %s", action, pth)
	}
}

func testBadRemove(action, pth string, isDir bool, e chan<- error) {
	if isDir {
		return
	}
	err := os.Remove(pth)
	if err == nil {
		e <- fmt.Errorf("%s:expected RO mount but could remove file: %s", action, pth)
	}
}

func testBadRename(action, pth string, _ bool, e chan<- error) {
	err := os.Rename(pth, sibling(pth, "renamed"))
	if err == nil {
		e <- fmt.Errorf("%s:expected RO mount but could rename file: %s", action, pth)
	}
}

func testBadSymlink(action, pth string, _ bool, e chan<- error) {
	err := os.Symlink(pth, sibling(pth, "linked"))
	if err == nil {
		e <- fmt.Errorf("%s:expected RO mount but could create symlink on file: %s", action, pth)
	}
}

func testOpenReadSeek(action, pth string, isDir bool, e chan<- error) {
	if isDir {
		return
	}

	file, err := os.Open(pth)
	if err != nil {
		e <- fmt.Errorf("%s:cannot open: %s: %w", action, pth, err)
		return
	}

	defer func() {
		err = file.Close()
		if err != nil {
			e <- fmt.Errorf("%s:cannot close: %s: %w", action, pth, err)
		}
	}()

	b := make([]byte, 10)
	_, err = file.Read(b)
	if err != nil {
		e <- fmt.Errorf("%s:cannot read: %s: %w", action, pth, err)
		return
	}

	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		e <- fmt.Errorf("%s:cannot seek: %s: %w", action, pth, err)
		return
	}

	_, err = file.ReadAt(b, 2)
	if err != nil {
		e <- fmt.Errorf("%s:cannot readAt: %s: %w", action, pth, err)
	}
}

func testBadOpenWrite(action, pth string, isDir bool, e chan<- error) {
	if isDir {
		return
	}

	file, err := os.OpenFile(pth, os.O_RDWR, 0755)
	if err != nil {
		return
	}
	e <- fmt.Errorf("%s:should not be able to open RDWR on RO mount: %s: %w", action, pth, err)

	// incidentally we get a file: let's continue trying with that one
	defer func() {
		err = file.Close()
		if err != nil {
			e <- fmt.Errorf("%s:cannot close: %s: %w", action, pth, err)
		}
	}()

	_, err = file.Write([]byte("test"))
	if err == nil {
		e <- fmt.Errorf("%s:expected RO mount but could overwrite file: %s", action, pth)
	}
}

func testBadOpenOverwrite(action, pth string, isDir bool, e chan<- error) {
	if isDir {
		return
	}

	file, err := os.OpenFile(pth, os.O_CREATE, 0755)
	// TODO: what it should be:
	/*
		if err != nil {
			return
		}
		e <- fmt.Errorf("%s:should not be able to open CREATE: %s", action, pth)
	*/
	// TODO: what it is now:
	if err != nil {
		e <- fmt.Errorf("%s:this is expected but not currently implemented: should not be able to open CREATE: %s", action, pth)
		return
	}

	// incidentally we get a file: let's continue trying with that one
	defer func() {
		err = file.Close()
		if err != nil {
			e <- fmt.Errorf("%s:cannot close: %s: %w", action, pth, err)
		}
	}()

	_, err = file.Write([]byte("test"))
	if err == nil {
		e <- fmt.Errorf("%s:expected RO mount but could overwrite file: %s", action, pth)
	}
}

func testBadOpenCreate(action, pth string, isDir bool, e chan<- error) {
	if isDir {
		return
	}
	_, err := os.OpenFile(sibling(pth, "new-file"), os.O_CREATE, 0755)
	if err == nil {
		e <- fmt.Errorf("%s:should not be able to create file: %w", action, err)
	}
}

func testStatFS(action, pth string, _ bool, e chan<- error) {
	var res syscall.Statfs_t
	err := syscall.Statfs(pth, &res) // TODO: fuse StatFS is not actually called
	if err != nil {
		e <- fmt.Errorf("%s:cannot statfs: %s: %w", action, pth, err)
	}
}

// TODO: file.Sync()
// TODO: file.WriteAt()
