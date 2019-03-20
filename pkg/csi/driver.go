package csi

type Driver struct {
	Config Config
}

type Config struct {
	Name    string
	Version string
}
