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
	// imapAddress = "mail.tolkeyenpatagonia.com:993"
	folderName = "INBOX"
	message_id = "<0c9701dabc0d$7aea8ec0$70bfac40$@inspiratravel.com>"
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
}
