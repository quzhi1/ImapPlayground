package main

import (
	"log"
	"os"

	"github.com/emersion/go-imap/client"
)

var imapAddress = "imap.mail.yahoo.com:993"

func main() {
	// Read auth
	username := os.Getenv("YAHOO_EMAIL_ADDRESS")
	password := os.Getenv("YAHOO_APP_PASSWORD")

	idle(username, password)
	log.Println("Done!")
}

func idle(username, password string) {
	// Connect to server
	c, err := client.DialTLS(imapAddress, nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected")

	// Login
	if err := c.Login(username, password); err != nil {
		log.Fatal(err)
	}
	log.Println("Logged in")

	// Don't forget to logout
	defer c.Logout()

	// Select folder
	_, err = c.Select("Inbox", true)
	if err != nil {
		log.Fatal(err)
	}

	// Create a channel to receive mailbox updates
	updates := make(chan client.Update)
	c.Updates = updates

	// Start idling
	stopped := false
	stop := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- c.Idle(stop, nil)
	}()

	// Listen for updates
	for {
		select {
		case update := <-updates:
			log.Println("New update:", update)
			if !stopped {
				close(stop)
				stopped = true
			}
		case err := <-done:
			if err != nil {
				log.Fatal(err)
			}
			log.Println("Not idling anymore")
			return
		}
	}
}
