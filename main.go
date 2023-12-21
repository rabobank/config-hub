package main

import (
	"fmt"
	"os"
	"path"

	"github.com/gomatbase/go-log"
	"github.com/rabobank/config-hub/cfg"
	"github.com/rabobank/config-hub/server"
)

const usage = "config-hub v%s\n\n" +
	"Usage:\n" +
	"  Server Mode:\n" +
	"    %s\n\n" +
	"  Credential Helper Mode:\n" +
	"    %s credentials <repo> <action>\n\n" +
	"    <action> = (get|store|erase)  Only \"get\" is processed\n\n" +
	"               It will read from Stdin a series of key/pair values from input lines in the format <key>=<value>\n" +
	"               An empty line triggers end of input and will process the command with the provided values.\n" +
	"               Credentials are provided (if available) when \"host\" and \"protocol\" are read from Stdin, as expected\n" +
	"               from git credential helpers. Store and erase actions will read key/values from stdin but are NOPs.\n" +
	"    <repo>   = The path component of the git repository url, this allows to fine tune the credentials requesst to\n" +
	"               specific repositories on the same host.\n"

func printUsage() int {
	name := path.Base(os.Args[0])
	fmt.Printf(usage, cfg.Version, name, name)
	return -1
}

func main() {
	cfg.Println("os.Args", os.Args)
	if len(os.Args) == 1 {
		fmt.Println("Logging level :", log.LevelName(cfg.LogLevel))
		server.Server()
	} else if len(os.Args) < 3 || os.Args[1] != "credentials" {
		os.Exit(printUsage())
	} else {
		cfg.FinishCredentialsConfiguration()
		if e := credentials(); e != nil {
			cfg.Println("error getting credentials", e)
			os.Exit(printUsage())
		}
	}
}
