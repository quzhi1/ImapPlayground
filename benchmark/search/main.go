package main

import (
	"log"
	"os"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// var imapAddress = "imap.mail.yahoo.com:993"

var imapAddress = "imap.mail.me.com:993"

func main() {
	// Read auth
	// username := os.Getenv("YAHOO_EMAIL_ADDRESS")
	// password := os.Getenv("YAHOO_APP_PASSWORD")
	username := os.Getenv("ICLOUD_EMAIL_ADDRESS")
	password := os.Getenv("ICLOUD_APP_PASSWORD")

	search(username, password)
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
	criteria := &imap.SearchCriteria{
		SentSince: time.Now().AddDate(0, 0, -30),
	}
	criteria.Not = []*imap.SearchCriteria{}
	// for _, exclude := range []string{"bank", "confidential"} {
	// 	subjectInclude := imap.SearchCriteria{Header: textproto.MIMEHeader{"Subject": {exclude}}}
	// 	criteria.Not = append(criteria.Not, &subjectInclude)
	// }
	criteria.Or = [][2]*imap.SearchCriteria{{
		{Text: []string{"order confirmation"}},
		&imap.SearchCriteria{
			Or: [][2]*imap.SearchCriteria{{
				{Text: []string{"order number"}},
				&imap.SearchCriteria{
					Or: [][2]*imap.SearchCriteria{{
						{Text: []string{"order information"}},
						{Text: []string{"order summary"}},
					}},
				},
			},
			},
		}}}
	uids, err := c.UidSearch(criteria)
	searchLatency := time.Now().UnixMilli() - start
	if err != nil {
		panic(err)
	}
	log.Printf("Search filter: %v, result: %v, latency: %d\n", criteria, uids, searchLatency)
	if len(uids) == 0 {
		log.Println("No message found")
		return
	}

	// Load messages
	log.Printf("Fetching message: %v\n", uids)
	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)
	// Setting peek to true will prevent marking the messages as read
	items := []imap.FetchItem{
		imap.FetchEnvelope,
	}
	messageChans := make(chan *imap.Message, len(uids))
	done := make(chan error, 1)
	go func() {
		done <- c.UidFetch(seqset, items, messageChans)
	}()
	if err := <-done; err != nil {
		panic(err)
	}

	// Print the messages
	for msg := range messageChans {
		if msg == nil {
			log.Printf("Server didn't returned message")
			break
		}
		log.Printf("MessageId: %s, subject: %s", msg.Envelope.MessageId, msg.Envelope.Subject)
	}
}
