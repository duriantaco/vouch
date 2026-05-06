package main

import (
	"os"

	"vouch/internal/vouch"
)

func main() {
	os.Exit(vouch.Main(os.Args[1:], os.Stdout, os.Stderr))
}
