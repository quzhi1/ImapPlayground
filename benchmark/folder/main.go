package main

import (
	"context"
	"crypto/tls"
	"os"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	username = os.Getenv("ICLOUD_EMAIL_ADDRESS")
	password = os.Getenv("ICLOUD_APP_PASSWORD")
)

const (
	// username    = "nylas.sdet@icloud.com"
	// password    = "assu-ndrx-lpzq-xjjg"
	imapAddress = "imap.mail.me.com:993"
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
	log.Ctx(ctx).Debug().Msgf("Connecting to IMAP server %s", imapAddress)
	imapClient, err := client.DialTLS(imapAddress, &tls.Config{InsecureSkipVerify: true}) //nolint:gosec // We support self signed imap server
	if err != nil {
		panic(err)
	}

	// Login
	if err := imapClient.Login(username, password); err != nil {
		panic(err)
	}
	log.Ctx(ctx).Debug().Str("username", username).Msg("Logged in to IMAP server")

	// List folders
	defer imapClient.Logout()
	mailboxesChan := make(chan *imap.MailboxInfo, 10)
	go imapClient.List("", "*", mailboxesChan)
	var folderStatus *imap.MailboxStatus
	for {
		select {
		case mailbox, ok := <-mailboxesChan:
			if ok {
				folderStatus, err = imapClient.Select(mailbox.Name, true)
				if err != nil {
					panic(err)
				}
				log.Ctx(ctx).Info().
					Str("folder", mailbox.Name).
					Strs("attributes", mailbox.Attributes).
					Uint32("uid_next", folderStatus.UidNext).
					Msg("Found folder")
			} else {
				return
			}
		default:
			continue
		}
	}
}
