package main

import (
	"context"
	"crypto/tls"
	"os"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	// username = os.Getenv("INTERMEDIA_EMAIL_ADDRESS")
	// password = os.Getenv("INTERMEDIA_PASSWORD")
	username = os.Getenv("ICLOUD_EMAIL_ADDRESS")
	password = os.Getenv("ICLOUD_APP_PASSWORD")
)

const (
	// imapAddress = "west.EXCH092.serverdata.net:993"
	imapAddress = "imap.mail.me.com:993"
	// imapAddress = "imap.1und1.de:993"
	folderName = "Papierkorb"
	message_id = "<CADPS7cRy8LrKiX2Zrf14x_rzo8VsCOGdM040ni_JphakRQHD6g@mail.gmail.com>"
)

func main() {
	// Init logger
	logger := zerolog.
		New(os.Stdout).
		With().
		Timestamp().
		Logger().
		Output(zerolog.ConsoleWriter{Out: os.Stderr})
	ctx := logger.WithContext(context.Background())

	// Connect to imap server
	imapClient, err := imapclient.DialTLS(imapAddress, &imapclient.Options{
		DebugWriter: os.Stdout,
		TLSConfig:   &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // We support self signed imap server
	})
	if err != nil {
		panic(err)
	}

	// Login
	if err := imapClient.Login(username, password).Wait(); err != nil {
		panic(err)
	}
	log.Ctx(ctx).Debug().Str("username", username).Msg("Logged in to IMAP server")

	// defer logout
	defer func() {
		if err := imapClient.Logout().Wait(); err != nil {
			panic(err)
		}
	}()

	// Select
	_, err = imapClient.Select(folderName, &imap.SelectOptions{
		ReadOnly: true,
	}).Wait()
	if err != nil {
		panic(err)
	}

	// Search for messages in the last 90 days
	criteria := imap.SearchCriteria{
		Header: []imap.SearchCriteriaHeaderField{
			{
				Key:   "Message-ID",
				Value: message_id,
			},
		},
	}
	log.Ctx(ctx).Debug().
		Str("folderName", folderName).
		Any("criteria", criteria).
		Msg("Searching folder")
	searchResponses, err := imapClient.UIDSearch(&criteria, nil).Wait()
	if err != nil {
		panic(err)
	}

	log.Ctx(ctx).Info().Any("uids", searchResponses.AllUIDs()).Msg("Found messages")
	if len(searchResponses.AllUIDs()) == 0 {
		log.Ctx(ctx).Warn().Msg("No messages found")
		return
	}

	// Fetch messages and print raw MIME into tmp.eml
	uid := searchResponses.AllUIDs()[0]
	log.Ctx(ctx).Info().Uint32("uid", uint32(uid)).Msg("Fetching message")
	seqSet := imap.UIDSetNum(uid)
	fetchOptions := &imap.FetchOptions{
		Envelope:    true,
		Flags:       true,
		UID:         true,
		BodySection: []*imap.FetchItemBodySection{{Peek: true}},
	}
	fetchCmd := imapClient.Fetch(seqSet, fetchOptions)
	fetchedMsgs, err := fetchCmd.Collect()
	if err != nil {
		panic(err)
	}
	if len(fetchedMsgs) == 0 {
		log.Ctx(ctx).Warn().Msg("No messages fetched")
		return
	}
}
