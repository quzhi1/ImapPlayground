package main

import (
	"log"
	"net/textproto"
	"os"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

var imapAddress = "imap.mail.yahoo.com:993"

func main() {
	// Read auth
	username := os.Getenv("YAHOO_EMAIL_ADDRESS")
	password := os.Getenv("YAHOO_APP_PASSWORD")

	search(username, password)
	log.Println("Done!")
}

func search(username, password string) {
	// Connect to server
	start := time.Now().UnixMilli()
	c, err := client.DialTLS(imapAddress, nil)
	if err != nil {
		log.Fatal(err)
	}
	connectLatency := time.Now().UnixMilli() - start
	log.Printf("Connected, latency: %d\n", connectLatency)

	// Login
	start = time.Now().UnixMilli()
	if err := c.Login(username, password); err != nil {
		log.Fatal(err)
	}
	loginLatency := time.Now().UnixMilli() - start
	log.Printf("Logged in, latency: %d\n", loginLatency)

	// Don't forget to logout
	defer c.Logout()

	// Select folder
	_, err = c.Select("Archive", true)
	if err != nil {
		log.Fatal(err)
	}

	// Search (from)
	start = time.Now().UnixMilli()
	searchCriteria := &imap.SearchCriteria{
		Header: textproto.MIMEHeader{
			"From": {"ivan@mail.notion.so"},
		},
	}
	uids, err := c.UidSearch(searchCriteria)
	searchLatency := time.Now().UnixMilli() - start
	if err != nil {
		panic(err)
	}
	log.Printf("Search filter: %v, result: %v, latency: %d\n", searchCriteria, uids, searchLatency)
}
