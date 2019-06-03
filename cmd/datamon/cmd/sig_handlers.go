// Copyright © 2018 One Concern

package cmd

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/jacobsa/fuse"
)

func registerSIGINTHandlerMount(mountPoint string) {
	// Register for SIGINT.
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	// Start a goroutine that will unmount when the signal is received.
	go func() {
		for {
			<-signalChan
			fmt.Println("Received SIGINT, attempting to unmount...")

			err := fuse.Unmount(mountPoint)
			if err != nil {
				fmt.Printf("Failed to unmount in response to SIGINT: %v", err)
			} else {
				fmt.Printf("Successfully unmounted in response to SIGINT.")
				return
			}
		}
	}()
}
