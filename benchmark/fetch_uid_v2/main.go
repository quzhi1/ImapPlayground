package main

import (
	"io"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"
)

const (
	// url = "imap.mail.me.com:993"
	url    = "ssl0.ovh.net:993"
	folder = "INBOX.Activité Recrutement.Cabinet LE NAIL"
	uid    = 62
)

func main() {
	// Read auth
	// username := os.Getenv("ICLOUD_EMAIL_ADDRESS")
	// password := os.Getenv("ICLOUD_APP_PASSWORD")
	username := os.Getenv("OVH_EMAIL_ADDRESS")
	password := os.Getenv("OVH_PASSWORD")

	// Connect
	option := &imapclient.Options{
		// DebugWriter: os.Stdout,
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

	// Select
	selectedMbox, err := c.Select(folder, &imap.SelectOptions{ReadOnly: true}).Wait()
	if err != nil {
		log.Fatalf("failed to select INBOX: %v", err)
	}
	log.Printf("%s contains %v messages", folder, selectedMbox.NumMessages)

	// Start fetch message command
	uid := imap.UIDSetNum(uid)
	fetchCmd := c.Fetch(uid, &imap.FetchOptions{
		Envelope:     true,
		Flags:        true,
		UID:          true,
		InternalDate: true,
		BodySection:  []*imap.FetchItemBodySection{{Peek: true}},
	})
	defer func() {
		if fetchCmd == nil {
			log.Println("fetchCmd is nil. Skipping closing.")
			return
		}
		if err := fetchCmd.Close(); err != nil {
			log.Println("Error closing fetch stream")
		}
	}()

	// Fetch message
	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		// Iterate over the fetched data items
		var envelope *imap.Envelope
		var flags []imap.Flag
		var uid *imap.UID
		var emlStr string
		var emlErr error
		var internalDate int64
		for {
			// Iterate over the fetched data items
			item := msg.Next()
			if item == nil {
				break
			}

			switch item := item.(type) {
			case imapclient.FetchItemDataEnvelope:
				envelope = item.Envelope
			case imapclient.FetchItemDataFlags:
				flags = item.Flags
			case imapclient.FetchItemDataUID:
				uid = &item.UID
			case imapclient.FetchItemDataInternalDate:
				internalDate = item.Time.Unix()
			case imapclient.FetchItemDataBodySection:
				// Why are we loading the entire eml string instead of passing the *mail.Reader?
				// Because *mail.Reader cannot be passed into `loadMessage` without EOF error.
				// I spent 2 days debugging this issue, still can't figure out why.
				// It is still OK to load the entire eml string into memory,
				// because message attachment will be at most 20 MB.
				var b []byte
				b, emlErr = io.ReadAll(item.Literal)
				emlStr = string(b)
			default:
				log.Printf("Unknown fetch item type : %s", reflect.TypeOf(item).String())
			}
		}

		// Skip message if there is an error reading the eml
		if emlErr != nil {
			log.Printf("error reading email eml, skipping this message: %v", emlErr)
			continue
		}
		log.Printf("flags: %v", flags)
		log.Printf("uid: %v", uid)
		log.Printf("internalDate: %d", internalDate)
		log.Printf("emlStr is empty? %t", emlStr == "")
		log.Printf("messageID from envelope: %s", envelope.MessageID)
		log.Printf("subject: %s", envelope.Subject)

		// Read the eml string
		emlStrReader := strings.NewReader(emlStr)
		var mailReader *mail.Reader
		mailReader, err := mail.CreateReader(emlStrReader)
		if err != nil {
			if message.IsUnknownCharset(err) {
				log.Fatalln("Unknown charset.")
			} else {
				log.Fatalln("Failed to create mail reader.")
			}
		}
		defer func() {
			if err = mailReader.Close(); err != nil {
				log.Fatalln("Failed to close mail reader")
			}
		}()

		messageIdFromRawMimie := mailReader.Header.Get("Message-ID")
		log.Printf("messageID from raw mime: %s", messageIdFromRawMimie)
	}
}
