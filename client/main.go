package main

import (
	"encoding/csv"
	"log"
	"net/textproto"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

var imapAddress = "imap.mail.yahoo.com:993"
var numOfTrials = 100
var mu sync.Mutex

func main() {
	// Read auth
	username := os.Getenv("YAHOO_EMAIL_ADDRESS")
	password := os.Getenv("YAHOO_APP_PASSWORD")

	// Init file
	fileName := "benchmark.csv"
	var csvFile *os.File
	if _, err := os.Stat(fileName); err != nil {
		csvFile, err = os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
	} else {
		csvFile, err = os.Create(fileName)
		if err != nil {
			panic(err)
		}
	}
	defer csvFile.Close()
	w := csv.NewWriter(csvFile)
	defer w.Flush()

	// Write
	w.Write([]string{
		"Connect IMAP server",
		"Login",
		// "List folders",
		"Counting messages in one folder",
		"List 10 messages",
		"Fetch message by uid",
		"Search messages with filter applied",
		"Move message from one folder to another",
		"Create folder",
		"Delete a folder",
	})

	// Multi-thread running benchmark
	// var wg sync.WaitGroup
	for i := 0; i < numOfTrials; i++ {
		// wg.Add(1)
		// go func(i int) {
		runFlow(username, password, w)
		// log.Printf("Trial %d done\n", i)
		// wg.Done()
		// }(i)
	}
	// wg.Wait()
	log.Println("Done!")
}

func runFlow(username, password string, writer *csv.Writer) {
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

	// // List mailboxes
	// start = time.Now().UnixMilli()
	// mailboxes := listFolder(c)
	// // listFolder(c)
	// listMbLatency := time.Now().UnixMilli() - start
	// // log.Println("Mailboxes:")
	// // for _, m := range mailboxes {
	// // 	log.Println("* " + m.Name)
	// // }
	// log.Printf("List mailbox. Number of folders: %d, latency: %d\n", len(mailboxes), listMbLatency)

	// Select Archive
	start = time.Now().UnixMilli()
	mbox, err := c.Select("Archive", false)
	if err != nil {
		log.Fatal(err)
	}
	selectLatency := time.Now().UnixMilli() - start
	log.Printf("Select folder \"Archive\". Number of messages: %d, latency: %d\n", mbox.Messages, selectLatency)

	// Get the last 10 messages
	start = time.Now().UnixMilli()
	messages := listMessage(c, mbox, 10)
	// listMessage(c, mbox, 10)
	listMsgLatency := time.Now().UnixMilli() - start
	// log.Println("Last 10 messages:")
	// for _, msg := range messages {
	// 	log.Printf("Subject: %s, Uid: %d\n", msg.Envelope.Subject, msg.Uid)
	// }
	log.Printf("List messages, num: %d, latency: %d\n", len(messages), listMsgLatency)

	// Read one message
	start = time.Now().UnixMilli()
	message := fetchMessageById(c, 190467)
	// fetchMessageById(c, 190467)
	fetchLatency := time.Now().UnixMilli() - start
	log.Printf("Fetch message. Subject: %s, latency: %d", message.Envelope.Subject, fetchLatency)

	// Search (only return uids)
	start = time.Now().UnixMilli()
	searchCriteria := &imap.SearchCriteria{
		Header: textproto.MIMEHeader{
			"From": {"ivan@mail.notion.so"},
		},
	}
	uids, err := c.UidSearch(searchCriteria)
	// _, err = c.UidSearch(searchCriteria)
	searchLatency := time.Now().UnixMilli() - start
	if err != nil {
		panic(err)
	}
	log.Printf("Search result: %v, latency: %d\n", uids, searchLatency)

	// Move a message from one folder to another
	mu.Lock()
	start = time.Now().UnixMilli()
	seqset := new(imap.SeqSet)
	seqset.AddNum(190461)
	err = c.UidMove(seqset, "INBOX")
	moveLatency := time.Now().UnixMilli() - start
	if err != nil {
		panic(err)
	}
	log.Printf("Message moved, latency: %d\n", moveLatency)

	// Move the message back
	err = c.UidMove(seqset, "Archive")
	if err != nil {
		panic(err)
	}
	mu.Unlock()

	// Create folder
	start = time.Now().UnixMilli()
	folderName := "CreatedByGo" + strconv.FormatInt(start, 10)
	err = c.Create(folderName)
	folderCreateLatency := time.Now().UnixMilli() - start
	if err != nil {
		panic(err)
	}
	log.Printf("Folder created, latency: %d\n", folderCreateLatency)

	// Delete folder
	start = time.Now().UnixMilli()
	err = c.Delete(folderName)
	folderDeleteLatency := time.Now().UnixMilli() - start
	if err != nil {
		panic(err)
	}
	log.Printf("Folder deleted, latency: %d\n", folderDeleteLatency)

	// Write csv
	writer.Write([]string{
		strconv.FormatInt(connectLatency, 10),
		strconv.FormatInt(loginLatency, 10),
		// strconv.FormatInt(listMbLatency, 10),
		strconv.FormatInt(selectLatency, 10),
		strconv.FormatInt(listMsgLatency, 10),
		strconv.FormatInt(fetchLatency, 10),
		strconv.FormatInt(searchLatency, 10),
		strconv.FormatInt(moveLatency, 10),
		strconv.FormatInt(folderCreateLatency, 10),
		strconv.FormatInt(folderDeleteLatency, 10),
	})
}

// Sometimes list folder got stuck

// func listFolder(client *client.Client) []imap.MailboxInfo {
// 	mailboxes := make(chan *imap.MailboxInfo, 10)
// 	done := make(chan error, 1)
// 	go func() {
// 		done <- client.List("", "*", mailboxes)
// 	}()

// 	if err := <-done; err != nil {
// 		log.Fatal(err)
// 	}

// 	result := []imap.MailboxInfo{}
// 	for m := range mailboxes {
// 		result = append(result, *m)
// 	}
// 	return result
// }

func listMessage(client *client.Client, folder *imap.MailboxStatus, lastNMessages uint32) []imap.Message {
	from := uint32(1)
	to := folder.Messages
	if folder.Messages > lastNMessages-1 {
		// We're using unsigned integers here, only subtract if the result is > 0
		from = folder.Messages - (lastNMessages - 1)
	}
	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- client.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope}, messages)
	}()
	if err := <-done; err != nil {
		log.Fatal(err)
	}

	result := []imap.Message{}
	for msg := range messages {
		result = append(result, *msg)
	}
	return result
}

func fetchMessageById(client *client.Client, id uint32) *imap.Message {
	seqset := new(imap.SeqSet)
	seqset.AddNum(id)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- client.UidFetch(seqset, []imap.FetchItem{imap.FetchEnvelope}, messages)
	}()
	if err := <-done; err != nil {
		log.Fatal(err)
	}

	for msg := range messages {
		if msg.Uid != id {
			log.Panicf("Message id does not match, actual: %d, expected: %d\n", msg.Uid, id)
		} else {
			return msg
		}
	}
	log.Panicf("No message with such id: %d\n", id)
	return nil
}
