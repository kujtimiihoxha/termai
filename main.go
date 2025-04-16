package main

import (
	"github.com/kujtimiihoxha/termai/cmd"
	"github.com/kujtimiihoxha/termai/internal/logging"
)

func main() {
	// Set up panic recovery for the main function
	defer logging.RecoverPanic("main", func() {
		// Perform any necessary cleanup before exit
		logging.ErrorPersist("Application terminated due to unhandled panic")
	})
	
	cmd.Execute()
}
