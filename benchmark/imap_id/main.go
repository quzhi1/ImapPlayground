package main

import (
	"log"
	"os"

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

	// ID
	idData, err := c.Id(&imap.IdData{
		Name:      "go-imap",
		Version:   "1.0",
		Os:        "Linux",
		OsVersion: "7.9.4",
		Vendor:    "Yahoo",
	}).Wait()
	if err != nil {
		panic(err)
	}
	log.Println(
		"Name:", idData.Name,
		"Version:", idData.Version,
		"Vendor:", idData.Vendor,
		"SupportUrl:", idData.SupportUrl,
	)
}
