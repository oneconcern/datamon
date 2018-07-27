package cflags

import units "github.com/docker/go-units"

// ByteSize used to pass byte sizes to a go-flags CLI
type ByteSize uint64

// String method for a bytesize (pflag value and stringer interface)
func (b ByteSize) String() string {
	return units.HumanSize(float64(b))
}

// Set the value of this bytesize (pflag value interfaces)
func (b *ByteSize) Set(value string) error {
	sz, err := units.FromHumanSize(value)
	if err != nil {
		return err
	}
	*b = ByteSize(uint64(sz))
	return nil
}

// Type returns the type of the pflag value (pflag value interface)
func (b *ByteSize) Type() string { return "byte-size" }
