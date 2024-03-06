package main

import (
	"context"
	"crypto/tls"
	"os"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	username = os.Getenv("INTERMEDIA_EMAIL_ADDRESS")
	password = os.Getenv("INTERMEDIA_PASSWORD")
	// username = os.Getenv("ICLOUD_EMAIL_ADDRESS")
	// password = os.Getenv("ICLOUD_APP_PASSWORD")
)

const (
	imapAddress = "west.EXCH092.serverdata.net:993"
	// imapAddress          = "imap.mail.me.com:993"
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
		// Select folder
		folder, err := imapClient.Select(folder.Name, true)
		if err != nil {
			panic(err)
		}
		log.Ctx(ctx).Debug().Str("folderName", folder.Name).Msg("Selected folder")

		// Search for messages in the last 90 days
		uids := searchOneFolder(ctx, imapClient)
		log.Ctx(ctx).Info().Str("folderName", folder.Name).Uints32("uids", uids).Msg("Found messages")

		// Load messages
		loadMsgs(ctx, imapClient, uids)
	}

	// Logout
	err = imapClient.Logout()
	log.Ctx(ctx).Debug().Msg("Logged out of IMAP server")
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Error logging out of IMAP server. We will directly close the connection")
	}
}

func listFolder(ctx context.Context, client *client.Client) []imap.MailboxInfo {
	mailboxes := make(chan *imap.MailboxInfo, 50)
	done := make(chan error, 1)
	go func() {
		done <- client.List("", "*", mailboxes)
	}()

	if err := <-done; err != nil {
		log.Ctx(ctx).Err(err).Msg("Error for listing folders")
	}

	var result []imap.MailboxInfo
	for m := range mailboxes {
		result = append(result, *m)
	}
	return result
}

func searchOneFolder(_ context.Context, imapClient *client.Client) []uint32 {
	criteria := &imap.SearchCriteria{
		SentSince: time.Now().AddDate(0, 0, -90),
	}
	uids, err := imapClient.UidSearch(criteria)
	if err != nil {
		panic(err)
	}
	return uids
}

func loadMsgs(ctx context.Context, imapClient *client.Client, uids []uint32) {
	if len(uids) == 0 {
		return
	}
	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)
	section := imap.BodySectionName{Peek: true}
	items := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchFlags,
		imap.FetchInternalDate,
		section.FetchItem(),
	}
	messageChans := make(chan *imap.Message, 10000)
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
		} else if msg.Envelope != nil {
			log.Ctx(ctx).Info().Time("date", msg.Envelope.Date).Str("message_id", msg.Envelope.MessageId).Msg("Got message")
		} else {
			log.Ctx(ctx).Warn().Msg("Envelope is nil")
		}

		r := msg.GetBody(&section)
		if r == nil {
			log.Ctx(ctx).Fatal().Msg("Server didn't returned message body")
		}

		log.Ctx(ctx).Info().Msgf("Subject: %s", msg.Envelope.Subject)
	}
}
