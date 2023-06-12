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
	username = os.Getenv("ICLOUD_EMAIL_ADDRESS_MANY_MESSAGES")
	password = os.Getenv("ICLOUD_APP_PASSWORD_MANY_MESSAGES")
)

const (
	imapAddress = "imap.mail.me.com:993"
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
	folders := listFolder(ctx, imapClient)
	for _, folder := range folders {
		log.Ctx(ctx).Info().Msgf("Found folder %s, flag %v", folder.Name, folder.Attributes)
	}

	// Select folder
	folder, err := imapClient.Select(folderName, true)
	if err != nil {
		panic(err)
	}
	log.Ctx(ctx).Debug().Str("folderName", folderName).Uint32("UIDVALIDITY", folder.UidValidity).Msg("Selected folder")

	// Get messages
	log.Ctx(ctx).Debug().Msgf("Fetching message: 1:421")
	seqset := new(imap.SeqSet)
	for i := 1; i <= 421; i++ {
		seqset.AddNum(uint32(i))
	}
	items := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchFlags,
	}
	messageChans := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- imapClient.UidFetch(seqset, items, messageChans)
	}()
	if err := <-done; err != nil {
		panic(err)
	}

	// Print the messages
	for msg := range messageChans {
		if msg == nil {
			log.Ctx(ctx).Fatal().Msg("Server didn't returned message")
		} else if msg.Envelope == nil {
			log.Ctx(ctx).Fatal().Msg("Server didn't returned message envelope")
		}
		log.Ctx(ctx).Info().Msgf("MessageId: %s", msg.Envelope.MessageId)
		log.Ctx(ctx).Info().Msgf("Date: %s", msg.Envelope.Date)
		log.Ctx(ctx).Info().Msgf("From: %v", msg.Envelope.From)
		log.Ctx(ctx).Info().Msgf("Sender: %v", msg.Envelope.Sender)
		log.Ctx(ctx).Info().Msgf("To: %v", msg.Envelope.To)
		log.Ctx(ctx).Info().Msgf("Cc: %v", msg.Envelope.Cc)
		log.Ctx(ctx).Info().Msgf("Bcc: %v", msg.Envelope.Bcc)
		log.Ctx(ctx).Info().Msgf("ReplyTo: %v", msg.Envelope.ReplyTo)
		log.Ctx(ctx).Info().Msgf("Subject: %s", msg.Envelope.Subject)

		log.Ctx(ctx).Info().Strs("flags", msg.Flags).Msg("List flags")
	}
	// Logout
	err = imapClient.Logout()
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Error logging out of IMAP server. We will directly close the connection")
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
