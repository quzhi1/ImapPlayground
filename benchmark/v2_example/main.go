package main

import (
	"log"
	"os"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

func main() {
	// Read auth
	// username := os.Getenv("YAHOO_EMAIL_ADDRESS")
	// password := os.Getenv("YAHOO_APP_PASSWORD")
	username := os.Getenv("ICLOUD_EMAIL_ADDRESS")
	password := os.Getenv("ICLOUD_APP_PASSWORD")

	// Connect
	option := &imapclient.Options{
		DebugWriter: os.Stdout,
	}
	// c, err := imapclient.DialTLS("imap.mail.yahoo.com:993", option)
	// c, err := imapclient.DialTLS("imap.mail.yahoo.com:993", nil)
	c, err := imapclient.DialTLS("imap.mail.me.com:993", option)
	if err != nil {
		log.Fatalf("failed to dial IMAP server: %v", err)
	}
	defer c.Close()

	// Login
	if err := c.Login(username, password).Wait(); err != nil {
		log.Fatalf("failed to login: %v", err)
	}

	// List mailboxes
	mailboxes, err := c.List("", "%", nil).Collect()
	if err != nil {
		log.Fatalf("failed to list mailboxes: %v", err)
	}
	log.Printf("Found %v mailboxes", len(mailboxes))
	for _, mbox := range mailboxes {
		log.Printf(" - %v", mbox.Mailbox)
	}

	// Select Archive
	selectedMbox, err := c.Select("Archive", &imap.SelectOptions{ReadOnly: true}).Wait()
	if err != nil {
		log.Fatalf("failed to select Archive: %v", err)
	}
	log.Printf("Archive contains %v messages", selectedMbox.NumMessages)

	// Fetch first message in Archive
	if selectedMbox.NumMessages > 0 {
		seqSet := imap.SeqSetNum(1)
		messages, err := c.Fetch(seqSet, &imap.FetchOptions{
			Envelope: true,
		}).Collect()
		if err != nil {
			log.Fatalf("failed to fetch first message in Archive: %v", err)
		}
		log.Printf("subject of first message in Archive: %v", messages[0].Envelope.Subject)
	}

	// Logout
	if err := c.Logout().Wait(); err != nil {
		log.Fatalf("failed to logout: %v", err)
	}
}
