package main

import (
	"embed"

	"github.com/fsncps/hyperfile/src/cmd"
)

var (
	//go:embed src/hyperfile_config/*
	content embed.FS
)

func main() {
	cmd.Run(content)
}
