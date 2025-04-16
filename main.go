package main

import (
	"github.com/kujtimiihoxha/opencode/cmd"
	"github.com/kujtimiihoxha/opencode/internal/logging"
)

func main() {
	defer logging.RecoverPanic("main", func() {
		logging.ErrorPersist("Application terminated due to unhandled panic")
	})

	cmd.Execute()
}
