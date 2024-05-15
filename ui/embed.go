package ui

import (
	"embed"
	"io/fs"
)

//go:embed dist/*
var uiFiles embed.FS

func UIFiles() fs.FS {
	fs, err := fs.Sub(uiFiles, "dist")
	if err != nil {
		panic("ui assets not found")
	}
	return fs
}
