package main

import (
	"flag"
	"github.com/oneconcern/datamon/pkg/driver"
	"log"
)

func main() {
	var (
		endpoint   = flag.String("endpoint", "", "CSI Endpoint")
		nodeID     = flag.String("nodeid", "", "node id")
		driverName = flag.String("drivername", "", "name of the driver")
		version    = flag.String("version", "", "Print the version and exit.")
	)

	flag.Parse()

	driver, err := driver.NewDriver(*nodeID, *version, *driverName, nil)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Datamon CSI driver going to run on endpoint: %s ", *endpoint)
	if err := driver.Run(*endpoint); err != nil {
		log.Fatal(err)
	}

}
