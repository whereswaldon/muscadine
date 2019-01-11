package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"path"
	"strings"

	"github.com/arborchat/muscadine/archive"
	"github.com/arborchat/muscadine/tui"
	"github.com/arborchat/muscadine/types"
)

// getDefaultLogFile returns a path to the default muscadine log file location.
func getDefaultLogFile() string {
	cwdFile := "muscadine.log"
	u, err := user.Current()
	if err != nil {
		return cwdFile
	}
	return path.Join(u.HomeDir, UserDataPath, cwdFile)
}

const serverAddressPlaceholder = "<server-address>"

// getDefaultHistFile returns a path to the default muscadine log file location.
func getDefaultHistFile(serverAddress string) string {
	return strings.Replace(getDefaultHistFileTemplate(), serverAddressPlaceholder, serverAddress, 1)
}

// getDefaultHistFileTemplate returns a path to the default muscadine log file location.
func getDefaultHistFileTemplate() string {
	cwdFile := serverAddressPlaceholder + ".arborhist"
	u, err := user.Current()
	if err != nil {
		return cwdFile
	}
	return path.Join(u.HomeDir, UserDataPath, cwdFile)
}

// configureLogging attempts to set the global logger to use the named file, and logs
// an error to stdout if it fails. It returns a teardown function that can be used to
// clean up the logging and print a status message to the user.
func configureLogging(logfile string) func() {
	if err := os.MkdirAll(path.Dir(logfile), 0700); err != nil {
		log.Printf("Error creating log storage directory (%s): %v\n", path.Dir(logfile), err)
	}
	file, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		log.Printf("Unable to begin logging to %s, %v", logfile, err)
		return func() {}
	}
	log.SetOutput(file)
	log.Println("--- New Session ---")
	return func() {
		fmt.Fprintln(os.Stderr, "Logs written to", file.Name())
		file.Close()
	}
}

func main() {
	var (
		ui                types.UI
		err               error
		username          string
		histfile, logfile string
		histfileTemplate  = getDefaultHistFileTemplate()
	)
	flag.StringVar(&username, "username", "muscadine", "Set your username on the server")
	flag.StringVar(&histfile, "histfile", histfileTemplate, "Load/Store history in this file")
	flag.StringVar(&logfile, "logfile", getDefaultLogFile(), "Write logs to this file")
	flag.Parse()
	if len(flag.Args()) < 1 {
		log.Fatal("Usage: " + os.Args[0] + " <ip>:<port>")
	}
	serverAddress := flag.Arg(0)
	if histfile == histfileTemplate {
		// use default history file
		histfile = getDefaultHistFile(serverAddress)
	}
	defer configureLogging(logfile)() // defer the returned cleanup function
	history, err := archive.NewManager(histfile)
	if err != nil {
		log.Fatalln("unable to construct archive", err)
	}
	if err := history.Load(); err != nil {
		log.Println("error loading history", err)
	}
	client, err := NewNetClient(serverAddress, username, history)
	if err != nil {
		log.Println("Error creating client", err)
		return
	}
	ui, err = tui.NewTUI(client)
	if err != nil {
		log.Fatal("Error creating TUI", err)
		return
	}
	ui.AwaitExit()
	if err := history.Save(); err != nil {
		log.Fatalln("error saving history", err)
	}
}
