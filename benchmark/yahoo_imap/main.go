package main

import (
	"context"
	"crypto/tls"
	"os"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-sasl"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	imapAddress = "imap.mail.yahoo.com:993"
	// imapAddress = "imap.mail.me.com:993"
	username    = "gma_imap_1@yahoo.com"
	accessToken = "" // Fill me
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

	// Create SASL client
	saslClient := sasl.NewOAuthBearerClient(&sasl.OAuthBearerOptions{
		Username: username,
		Token:    accessToken,
	})

	// Login
	if err := imapClient.Authenticate(saslClient); err != nil {
		panic(err)
	}
	log.Ctx(ctx).Debug().Str("username", username).Msg("Logged in to IMAP server")

	// List folders
	folders := listFolder(ctx, imapClient)
	for _, folder := range folders {
		log.Ctx(ctx).Info().Msgf("Found folder %s, flag %v", folder.Name, folder.Attributes)
	}
}

func listFolder(ctx context.Context, client *client.Client) []imap.MailboxInfo {
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- client.List("", "*", mailboxes)
	}()

	if err := <-done; err != nil {
		log.Ctx(ctx).Err(err).Msg("Error for listing folders")
	}

	result := []imap.MailboxInfo{}
	for m := range mailboxes {
		result = append(result, *m)
	}
	return result
}
