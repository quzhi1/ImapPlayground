package main

import (
	"log"
	"os"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

func main() {
	// Read auth
	username := os.Getenv("YAHOO_EMAIL_ADDRESS")
	password := os.Getenv("YAHOO_APP_PASSWORD")

	// Connect
	// option := &imapclient.Options{
	// 	DebugWriter: os.Stdout,
	// }
	// c, err := imapclient.DialTLS("imap.mail.yahoo.com:993", option)
	c, err := imapclient.DialTLS("imap.mail.yahoo.com:993", nil)
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

	// Select INBOX
	selectedMbox, err := c.Select("INBOX").Wait()
	if err != nil {
		log.Fatalf("failed to select INBOX: %v", err)
	}
	log.Printf("INBOX contains %v messages", selectedMbox.NumMessages)

	// Fetch first message in INBOX
	if selectedMbox.NumMessages > 0 {
		seqSet := imap.SeqSetNum(1)
		fetchItems := []imap.FetchItem{imap.FetchItemEnvelope}
		messages, err := c.Fetch(seqSet, fetchItems).Collect()
		if err != nil {
			log.Fatalf("failed to fetch first message in INBOX: %v", err)
		}
		log.Printf("subject of first message in INBOX: %v", messages[0].Envelope.Subject)
	}

	// Logout
	if err := c.Logout().Wait(); err != nil {
		log.Fatalf("failed to logout: %v", err)
	}
}
