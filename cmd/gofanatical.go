package main

import (
	"log/slog"
	"os"

	gofanatical "github.com/Feuerlord2/Fanatical-RSS-Site/pkg"
)

func main() {
	if err := gofanatical.Run(); err != nil {
		slog.Error("feed generation failed", "error", err)
		os.Exit(1)
	}
}
