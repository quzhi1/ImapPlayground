package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

func main() {
	option := &imapclient.Options{
		DebugWriter: os.Stdout,
	}

	// Connect
	c, err := imapclient.DialTLS("imap.mail.yahoo.com:993", option)
	// c, err := imapclient.DialTLS("imap.mail.me.com:993", option)
	if err != nil {
		panic(err)
	}

	// Get os name and version
	osName, err := getOSName()
	if err != nil {
		panic(err)
	}
	osVersion, err := getOSVersion()
	if err != nil {
		panic(err)
	}

	// ID
	idData, err := c.ID(&imap.IDData{
		Name:        "go-imap",
		Version:     "v3.0",
		OS:          osName,
		OSVersion:   osVersion,
		Vendor:      "Nylas",
		SupportURL:  "https://support.nylas.com/",
		Address:     "2100 Geng Rd. #2100, Palo Alto, CA 94303",
		Date:        time.Now().Format("2-Jan-2006"),
		Command:     "go run benchmark/imap_id/main.go",
		Arguments:   "",
		Environment: "development",
	}).Wait()
	if err != nil {
		panic(err)
	}
	log.Println(
		"Name:", idData.Name,
		"Version:", idData.Version,
		"Vendor:", idData.Vendor,
		"SupportUrl:", idData.SupportURL,
	)
}

// getOSName retrieves the OS name based on the operating system.
func getOSName() (string, error) {
	switch runtime.GOOS {
	case "windows":
		return "Windows", nil
	case "darwin":
		return "macOS", nil
	case "linux":
		return getLinuxOSName()
	default:
		return "", fmt.Errorf("unsupported platform")
	}
}

// getLinuxOSName reads the /etc/os-release file to determine the Linux distribution name.
func getLinuxOSName() (string, error) {
	cmd := exec.Command("cat", "/etc/os-release")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "NAME=") {
			return strings.Trim(line[5:], "\""), nil
		}
	}

	return "", fmt.Errorf("could not determine Linux distribution name")
}

func getOSVersion() (string, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "ver")
	case "darwin":
		cmd = exec.Command("sw_vers", "-productVersion")
	case "linux":
		cmd = exec.Command("cat", "/etc/os-release")
	default:
		return "", fmt.Errorf("unsupported platform")
	}

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}
