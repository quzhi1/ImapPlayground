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
	// username := os.Getenv("ICLOUD_EMAIL_ADDRESS")
	// password := os.Getenv("ICLOUD_APP_PASSWORD")
	// username := os.Getenv("MCSPOWERMAIL_EMAIL_ADDRESS")
	// password := os.Getenv("MCSPOWERMAIL_PASSWORD")
	username := os.Getenv("STARTMAIL_EMAIL_ADDRESS")
	password := os.Getenv("STARTMAIL_PASSWORD")
	// username := os.Getenv("SITEGROUND_EMAIL_ADDRESS")
	// password := os.Getenv("SITEGROUND_PASSWORD")

	// url := "imap.mail.me.com:993"
	// url := "mail.mcspowermail.com:993"
	url := "imap.startmail.com:993"
	// url := "uk49.siteground.eu:993"

	// Connect
	option := &imapclient.Options{
		DebugWriter: os.Stdout,
	}
	c, err := imapclient.DialTLS(url, option)
	if err != nil {
		log.Fatalf("failed to dial IMAP server: %v", err)
	}
	defer c.Close()

	// Login
	if err := c.Login(username, password).Wait(); err != nil {
		log.Fatalf("failed to login: %v", err)
	}

	// Defer logout
	defer func() {
		if err := c.Logout().Wait(); err != nil {
			log.Fatalf("failed to logout: %v", err)
		}
	}()

	// List mailboxes
	mailboxes, err := c.List("", "*", nil).Collect()
	if err != nil {
		log.Fatalf("failed to list mailboxes: %v", err)
	}
	log.Printf("Found %v mailboxes", len(mailboxes))
	for _, mbox := range mailboxes {
		log.Printf(" - %v", mbox.Mailbox)
	}

	// Select INBOX
	selectedMbox, err := c.Select("INBOX", &imap.SelectOptions{ReadOnly: true}).Wait()
	// selectedMbox, err := c.Select("INBOX", nil).Wait()
	if err != nil {
		log.Fatalf("failed to select INBOX: %v", err)
	}
	log.Printf("INBOX contains %v messages", selectedMbox.NumMessages)

	// Fetch first message in INBOX
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

	// Check imap client state
	log.Println("IMAP client state:", c.State())
}
