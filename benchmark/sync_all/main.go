package main

import (
	"context"
	"crypto/tls"
	"io"
	"os"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/mail"
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
	HTMLContentType      = "text/html"
	PlainTextContentType = "text/plain"
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
		TLSConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // We support self signed imap server
	})
	if err != nil {
		panic(err)
	}

	// Login
	if err := imapClient.Login(username, password).Wait(); err != nil {
		panic(err)
	}
	log.Ctx(ctx).Debug().Str("username", username).Msg("Logged in to IMAP server")

	// List folders
	folders, err := imapClient.List("", "*", nil).Collect()
	if err != nil {
		panic(err)
	}

	for _, folder := range folders {
		// Select folder
		_, err := imapClient.Select(folder.Mailbox, &imap.SelectOptions{
			ReadOnly: true,
		}).Wait()
		if err != nil {
			panic(err)
		}

		// Search for messages in the last 7 days
		uids := searchOneFolder(ctx, imapClient, folder.Mailbox)
		log.Ctx(ctx).Info().Str("folderName", folder.Mailbox).Any("uids", uids).Msg("Found messages")

		// Load message
		loadMsg(ctx, imapClient, uids)
	}

	// Logout
	err = imapClient.Logout().Wait()
	log.Ctx(ctx).Debug().Msg("Logged out of IMAP server")
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Error logging out of IMAP server. We will directly close the connection")
	}
}

func searchOneFolder(_ context.Context, imapClient *imapclient.Client, folderName string) []imap.UID {
	// Select folder
	_, err := imapClient.Select(folderName, &imap.SelectOptions{ReadOnly: true}).Wait()
	if err != nil {
		panic(err)
	}
	// log.Ctx(ctx).Debug().Str("folderName", folderName).Msg("Searching folder")

	// Search for messages in the last 90 days
	criteria := imap.SearchCriteria{
		SentSince: time.Now().AddDate(0, 0, -90),
	}
	searchResponses, err := imapClient.UIDSearch(&criteria, nil).Wait()
	if err != nil {
		panic(err)
	}

	return searchResponses.AllUIDs()
}

func loadMsg(ctx context.Context, imapClient *imapclient.Client, uids []imap.UID) {
	if len(uids) == 0 {
		return
	}

	// Send a FETCH command to fetch the message body
	seqSet := imap.UIDSetNum(uids...)
	fetchOptions := &imap.FetchOptions{
		Envelope:     true,
		Flags:        true,
		InternalDate: true,
		BodySection:  []*imap.FetchItemBodySection{{}},
	}
	fetchCmd := imapClient.Fetch(seqSet, fetchOptions)
	defer fetchCmd.Close()

	// Find the body section in the response
	var bodySection imapclient.FetchItemDataBodySection
	var ok bool
	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}
		for {
			item := msg.Next()
			if item == nil {
				break
			}
			bodySection, ok = item.(imapclient.FetchItemDataBodySection)
			if ok {
				readBodySection(ctx, bodySection)
			}

		}
	}

	if err := fetchCmd.Close(); err != nil {
		panic(err)
	}
}

func readBodySection(ctx context.Context, bodySection imapclient.FetchItemDataBodySection) {
	// Read the message via the go-message library
	mr, err := mail.CreateReader(bodySection.Literal)
	if err != nil {
		panic(err)
	}

	// Print a few header fields
	h := mr.Header
	if subject, err := h.Text("Subject"); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to parse Subject header field")
	} else {
		log.Ctx(ctx).Info().Str("subject", subject).Msg("Loading message")
	}

	// Process the message's parts
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}

		switch p.Header.(type) {
		case *mail.InlineHeader:
			// This is the message's text (can be plain-text or HTML)
			// b, _ := io.ReadAll(p.Body)
			// log.Ctx(ctx).Info().Str("text", string(b)).Msg("body")
		}
	}
}
