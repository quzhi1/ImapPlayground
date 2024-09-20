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
	// username = os.Getenv("ICLOUD_EMAIL_ADDRESS")
	// password = os.Getenv("ICLOUD_APP_PASSWORD")
	username = os.Getenv("CHINESE_263_EMAIL_ADDRESS")
	password = os.Getenv("CHINESE_263_PASSWORD")
)

const (
	// imapAddress = "west.EXCH092.serverdata.net:993"
	// imapAddress = "imap.mail.me.com:993"
	imapAddress = "imapw.263.net:993"
	folderName  = "INBOX"
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
		DebugWriter: os.Stderr,
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

	// Select
	_, err = imapClient.Select(folderName, &imap.SelectOptions{
		ReadOnly: true,
	}).Wait()
	if err != nil {
		panic(err)
	}

	// Search for messages in the last 90 days
	criteria := imap.SearchCriteria{
		// SentSince: time.Now().AddDate(0, 0, -90),
	}
	// log.Ctx(ctx).Debug().
	// 	Str("folderName", folderName).
	// 	Any("criteria", criteria).
	// 	Msg("Searching folder")
	searchResponses, err := imapClient.UIDSearch(&criteria, nil).Wait()
	// searchResponses, err := imapClient.UIDSearch(nil, nil).Wait()
	if err != nil {
		panic(err)
	}

	log.Ctx(ctx).Info().Any("uids", searchResponses.AllUIDs()).Msg("Found messages")

	// Logout
	if err := imapClient.Logout().Wait(); err != nil {
		panic(err)
	}
}
