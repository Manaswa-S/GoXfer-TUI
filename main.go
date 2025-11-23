package main

import (
	"goxfer/tui/cipher"
	"goxfer/tui/core"
	"goxfer/tui/stages/auxiliary"
	"goxfer/tui/stages/router"

	"github.com/rivo/tview"
)

func main() {
	app := tview.NewApplication()

	cipher := cipher.NewAESGSMCipher()

	core, err := core.NewCore("http://localhost:9090/api/v1/", cipher) // TODO: domain should be dynamic
	if err != nil {
		panic(err)
	}

	settings := auxiliary.NewSettings()
	err = settings.InitSettings()
	if err != nil {
		panic(err)
	}

	allStages := router.NewStages(app, core, cipher, settings)
	pages, err := allStages.InitStages()
	if err != nil {
		panic(err)
	}

	if err := app.SetRoot(pages, true).Run(); err != nil {
		panic(err)
	}
}

func init() {
	// TODO: verify program checksums and all then pull all configs from server.
}
