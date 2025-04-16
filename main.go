package main

import (
	"github.com/kujtimiihoxha/opencode/cmd"
	"github.com/kujtimiihoxha/opencode/internal/logging"
)

func main() {
	// Set up panic recovery for the main function
	defer logging.RecoverPanic("main", func() {
		// Perform any necessary cleanup before exit
		logging.ErrorPersist("Application terminated due to unhandled panic")
	})
	
	cmd.Execute()
}
