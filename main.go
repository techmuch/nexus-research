package main

import (
	"embed"

	"github.com/techmuch/nexus-research/cmd"
)

//go:embed all:frontend/dist
var frontendFS embed.FS

func main() {
	cmd.SetFrontendFS(frontendFS)
	cmd.Execute()
}
