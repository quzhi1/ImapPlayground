package main

import (
	"log"
	"os"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

func main() {
	// Read auth
	// username := os.Getenv("YAHOO_EMAIL_ADDRESS")
	// password := os.Getenv("YAHOO_APP_PASSWORD")
	// username := os.Getenv("ICLOUD_EMAIL_ADDRESS")
	// password := os.Getenv("ICLOUD_APP_PASSWORD")
	username := os.Getenv("INTERMEDIA_EMAIL_ADDRESS")
	password := os.Getenv("INTERMEDIA_PASSWORD")

	// Connect
	option := &imapclient.Options{
		DebugWriter: os.Stdout,
		UnilateralDataHandler: &imapclient.UnilateralDataHandler{
			Expunge: func(seqNum uint32) {
				log.Printf("message %v has been expunged", seqNum)
			},
			Mailbox: func(data *imapclient.UnilateralDataMailbox) {
				if data.NumMessages != nil {
					log.Printf("a new message has been received")
				}
			},
			Fetch: func(msg *imapclient.FetchMessageData) {
				log.Printf("message %v got changed", msg.SeqNum)
			},
		},
	}
	// c, err := imapclient.DialTLS("imap.mail.yahoo.com:993", option)
	// c, err := imapclient.DialTLS("imap.mail.yahoo.com:993", nil)
	// c, err := imapclient.DialTLS("imap.mail.me.com:993", option)
	c, err := imapclient.DialTLS("west.EXCH092.serverdata.net:993", option)
	if err != nil {
		log.Fatalf("failed to dial IMAP server: %v", err)
	}
	defer c.Close()

	// Login
	if err := c.Login(username, password).Wait(); err != nil {
		log.Fatalf("failed to login: %v", err)
	}

	// Select INBOX
	selectedMbox, err := c.Select("INBOX", &imap.SelectOptions{ReadOnly: true}).Wait()
	if err != nil {
		log.Fatalf("failed to select Archive: %v", err)
	}
	log.Printf("INBOX contains %v messages", selectedMbox.NumMessages)

	for {
		// Start idling every 30 seconds
		idleCmd, err := c.Idle()
		if err != nil {
			log.Printf("IDLE command failed: %v, retrying\n", err)
			continue
		}

		// Wait for 30s
		time.Sleep(30 * time.Second)

		// Stop idling
		if err := idleCmd.Close(); err != nil {
			log.Fatalf("failed to stop idling: %v", err)
		}
	}
}
