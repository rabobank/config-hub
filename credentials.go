package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gomatbase/csn"
	"github.com/rabobank/config-hub/cfg"
	"github.com/rabobank/config-hub/domain"
	"github.com/rabobank/config-hub/util"
)

const (
	UnknownActionError          = csn.ErrorF("Unknown Action: $s")
	ExpectedHosAndProtocolError = csn.Error("Expected both host and protocol")
)

func readProperties() map[string]string {
	properties := make(map[string]string)
	input := bufio.NewScanner(os.Stdin)

	// ignore the error, we'll just stop reading
	for input.Scan() {
		line := input.Text()

		if len(line) == 0 {
			break
		}

		split := strings.Index(line, "=")
		properties[line[:split]] = line[split+1:]
	}
	return properties
}

func isKeyPresent(properties map[string]string, key string) bool {
	_, found := properties[key]
	return found
}

func Credentials() error {
	action := os.Args[len(os.Args)-1]
	cfg.Println("Calling credentials action :", action)
	switch action {
	case "get":
		cfg.Println("Reading properties")
		properties := readProperties()
		cfg.Println("properties :", properties)
		if !isKeyPresent(properties, "host") || !isKeyPresent(properties, "protocol") {
			return ExpectedHosAndProtocolError
		}

		request := &domain.CredentialsRequest{
			Protocol: properties["protocol"],
			Host:     properties["host"],
			Repo:     os.Args[len(os.Args)-2],
		}
		response := &domain.HttpCredentials{}
		if e := util.Request("http://localhost:8080", "credentials").PostJsonExchange(&request, &response); e != nil {
			return e
		}
		// credential.helper is only used for https urls
		protocol := fmt.Sprintf("protocol=%s", request.Protocol)
		host := fmt.Sprintf("host=%s", request.Host)
		username := fmt.Sprintf("username=%s", response.Username)
		password := fmt.Sprintf("password=%s", response.Password)
		fmt.Println(protocol)
		cfg.Println(protocol)
		fmt.Println(host)
		cfg.Println(host)
		fmt.Println(username)
		cfg.Println(username)
		fmt.Println(password)
		cfg.Println(password)
	case "store":
		readProperties()
	case "erase":
		readProperties()
	default:
		return UnknownActionError.WithValues(action)
	}

	return nil
}
