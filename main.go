package main

import (
	"os"

	"github.com/aaronflorey/genignore/internal/app"
)

func main() {
	exitCode := app.Run(os.Args[1:])
	os.Exit(exitCode)
}
