package main

import (
	"context"
	"crypto/tls"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-sasl"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
)

const (
	imapAddress = "imap.mail.yahoo.com:993"
	username    = "nylasinc@yahoo.com"
	accessToken = ""
)

var password = os.Getenv("NYLAS_INC_YAHOO_APP_PASSWORD")

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
	imapClient, err := client.DialTLS(imapAddress, &tls.Config{InsecureSkipVerify: true}) //nolint:gosec // We support self-signed imap server
	if err != nil {
		panic(err)
	}

	// Create SASL client
	saslClient := sasl.NewOAuthBearerClient(&sasl.OAuthBearerOptions{
		Username: username,
		Token:    accessToken,
	})

	// Login
	err = imapClient.Authenticate(saslClient)
	//err = imapClient.Login(username, password)
	if err != nil {
		panic(err)
	}

	log.Ctx(ctx).Debug().Str("username", username).Msg("Logged in to IMAP server")

	// Defer logout
	defer func() {
		if err = imapClient.Logout(); err != nil {
			log.Ctx(ctx).Fatal().Err(err)
		}
	}()

	// Select INBOX
	_, err = imapClient.Select("INBOX", false)
	if err != nil {
		log.Ctx(ctx).Fatal().Err(err)
	}

	// List some uids
	criteria := imap.NewSearchCriteria()
	criteria.Header.Add("Subject", "Zhi Test")
	uids, err := imapClient.UidSearch(criteria)
	if err != nil {
		log.Ctx(ctx).Fatal().Err(err)
	}

	if len(uids) == 0 {
		log.Ctx(ctx).Fatal().Msg("No message in inbox")
	}

	uid := uids[0]
	log.Ctx(ctx).Info().Uint32("uid", uid).Msg("Select the first uid")
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uid)

	// Set the flag
	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.FlaggedFlag}
	err = imapClient.UidStore(seqSet, item, flags, nil)
	if err != nil {
		log.Ctx(ctx).Fatal().Err(err)
	}
	log.Ctx(ctx).Info().Msg("Message changed")

	// Verify
	done := make(chan error, 1)
	messageChans := make(chan *imap.Message, 10)
	items := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchFlags,
	}
	go func() {
		done <- imapClient.UidFetch(seqSet, items, messageChans)
	}()
	if err := <-done; err != nil {
		panic(err)
	}

	// Print the messages
	for msg := range messageChans {
		log.Ctx(ctx).Info().Strs("flags", msg.Flags).Str("subject", msg.Envelope.Subject).Msg("Reading message")
	}
}
