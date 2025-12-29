package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"goxfer/tui/cipher"
	"goxfer/tui/consts"
	"goxfer/tui/core"
	"goxfer/tui/logger"
	"goxfer/tui/logger/native"
	"goxfer/tui/stages/auxiliary"
	"goxfer/tui/stages/router"
	"net/http"
	"time"

	"github.com/rivo/tview"
)

/*
	All stdin, stdout and stderr operations should happen in here only.
	That is to prevent gibberish to be printed as Application.Run owns them all during event loop's
	runtime.
*/

func main() {
	fmt.Println("Starting GoXfer ... ")

	sessBytes := make([]byte, 32)
	_, err := rand.Read(sessBytes)
	if err != nil {
		fmt.Printf("failed to read random: %v\n", err)
		return
	}
	sessionId := base64.StdEncoding.EncodeToString(sessBytes)
	fmt.Printf("Session ID: %s\nThis ID is not sensitive and is only used for debugging purposes.\n", sessionId)

	logger, err := newLogger(sessionId)
	if err != nil {
		fmt.Printf("failed to get new logger: %v\n", err)
		return
	}
	defer logger.Stop()

	//
	app := tview.NewApplication()
	defer app.Stop()

	cipher := cipher.NewAESGSMCipher()

	core, err := core.NewCore("http://localhost:9090/api/v1/", cipher) // TODO: domain should be dynamic
	if err != nil {
		fmt.Printf("failed to get new core: %v\n", err)
		return
	}

	settings, err := auxiliary.NewSettings()
	if err != nil {
		fmt.Printf("failed to get settings: %v\n", err)
		return
	}

	err = checkConn(core)
	if err != nil {
		fmt.Printf("failed to connect to the server: %v\n", err)
		return
	}

	allStages := router.NewStages(logger, app, core, cipher, settings)
	pages, err := allStages.InitStages()
	if err != nil {
		fmt.Printf("failed to init stages: %v\n", err)
		return
	}

	if err := app.SetRoot(pages, true).Run(); err != nil {
		fmt.Printf("failed to set root: %v\n", err)
		return
	}

	time.Sleep(100 * time.Millisecond)
	fmt.Println("Stopping GoXfer ... ")
}

func newLogger(sessionId string) (logger.Logger, error) {
	logger, err := native.New(consts.LOGS_FILE_PATH, consts.LOGS_MAX_FILE_SIZE, consts.LOGS_MAX_TIME)
	if err != nil {
		return nil, err
	}
	logger.With(sessionId)
	return logger, nil
}

func init() {
	// TODO: verify program checksums and all then pull all configs from server.
}

func checkConn(coreS *core.Core) error {
	resp, _, err := coreS.Hit(core.Routes.Welcome, nil, nil, nil)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status not ok")
	}

	return nil
}
