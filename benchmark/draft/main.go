package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

var (
	username = os.Getenv("ICLOUD_EMAIL_ADDRESS")
	password = os.Getenv("ICLOUD_APP_PASSWORD")
)

const (
	imapAddress = "imap.mail.me.com:993"
)

func main() {
	// Connect to the IMAP server
	c, err := client.DialTLS(imapAddress, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Logout()

	// Login to the iCloud account
	if err := c.Login(username, password); err != nil {
		log.Fatal(err)
	}

	// Select the mailbox
	_, err = c.Select("Drafts", false)
	if err != nil {
		log.Fatal(err)
	}

	// Create an IMAP message from the mail.Message
	msgBytes := createMessage()

	// Append the email to the Drafts mailbox
	err = c.Append("Drafts", nil, time.Now(), msgBytes)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Email draft created successfully.")
}

func createMessage() *bytes.Buffer {
	var b bytes.Buffer
	var h mail.Header
	h.SetSubject("Test subject")
	mw, err := mail.CreateWriter(&b, h)
	if err != nil {
		log.Fatal(err)
	}

	// Create a text part
	tw, err := mw.CreateInline()
	if err != nil {
		log.Fatal(err)
	}
	var th mail.InlineHeader
	th.Set("Content-Type", "text/plain")
	w, err := tw.CreatePart(th)
	if err != nil {
		log.Fatal(err)
	}
	io.WriteString(w, "Who are you?")
	w.Close()
	tw.Close()

	// Create an attachment
	var ah mail.AttachmentHeader
	ah.Set("Content-Type", "text/plain")
	ah.SetFilename("note.txt")
	w, err = mw.CreateAttachment(ah)
	if err != nil {
		log.Fatal(err)
	}
	io.WriteString(w, "Attachment content")
	w.Close()

	mw.Close()

	return &b
}
