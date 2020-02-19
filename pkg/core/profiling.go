package core

import (
	"os"
	"path/filepath"
	"runtime/pprof"
)

func writeMemProfile(opts ...Option) error {
	settings := newSettings(opts...)
	if dir := settings.memProfDir; dir != "" {
		path := filepath.Join(dir, "upload_bundle.mem.prof")
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		err = pprof.Lookup("heap").WriteTo(f, 0)
		if err != nil {
			return err
		}
	}
	return nil
}
