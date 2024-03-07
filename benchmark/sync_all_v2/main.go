package main

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"
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
	imapAddress          = "imap.mail.me.com:993"
	HTMLContentType      = "text/html"
	PlainTextContentType = "text/plain"
	multipartError       = "multipart:"
	encodingError        = "encoding error"
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
		loadMsgs(ctx, imapClient, uids)
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

func loadMsgs(ctx context.Context, imapClient *imapclient.Client, uids []imap.UID) {
	if len(uids) == 0 {
		return
	}

	// Send a FETCH command to fetch the message body
	seqSet := imap.UIDSetNum(uids...)
	fetchOptions := &imap.FetchOptions{
		Envelope:    true,
		Flags:       true,
		UID:         true,
		BodySection: []*imap.FetchItemBodySection{{Peek: true}},
	}
	fetchCmd := imapClient.Fetch(seqSet, fetchOptions)
	defer fetchCmd.Close()

	// Find the body section in the response
	var mr *mail.Reader
	var err error
	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}
		for {
			// Iterate over the fetched data items
			item := msg.Next()
			if item == nil {
				break
			}

			switch item := item.(type) {
			case imapclient.FetchItemDataEnvelope:
				// log.Ctx(ctx).Debug().Any("from", item.Envelope.From).Msg("Reading envelope")
			case imapclient.FetchItemDataBodySection:
				// Read the message via the go-message library
				mr, err = mail.CreateReader(item.Literal)
				if err != nil {
					panic(err)
				}
			case imapclient.FetchItemDataFlags:
				// log.Ctx(ctx).Debug().Any("flags", item.Flags).Msg("Reading flags")
			case imapclient.FetchItemDataUID:
				// log.Ctx(ctx).Debug().Uint32("uid", uint32(item.UID)).Msg("Reading UID")
			default:
				log.Ctx(ctx).Warn().Str("fetch_item_type", reflect.TypeOf(item).String()).Msg("Unknown fetch item type")
			}
		}
	}
	parseBody(ctx, mr)
}

func parseBody(ctx context.Context, mr *mail.Reader) {
	for {
		p, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			shouldBreak := false
			switch {
			case message.IsUnknownCharset(err):
				log.Ctx(ctx).Warn().Err(err).Msg("Ignore unknown charset")
			case strings.Contains(err.Error(), "malformed MIME header line"):
				log.Ctx(ctx).Warn().Err(err).Msg("Ignore malformed MIME header line")
			case strings.Contains(err.Error(), multipartError):
				log.Ctx(ctx).Warn().Err(err).Msg("Ignore multipart error")
				shouldBreak = true
			case strings.Contains(err.Error(), encodingError):
				log.Ctx(ctx).Warn().Err(err).Msg("Ignore encoding error")
				shouldBreak = true
			default:
				log.Ctx(ctx).Error().Err(err).Msg("Error reading message part")
				panic(err)
			}

			if shouldBreak {
				break
			}
		}

		if p == nil {
			continue
		}

		switch partHeader := p.Header.(type) {
		case *mail.InlineHeader:
			// This is the message's text (can be plain-text or HTML)
			contentType := partHeader.Header.Header.Get("Content-Type")
			log.Ctx(ctx).Debug().Str("content_type", contentType).Msg("found content-type")
			switch {
			case strings.Contains(contentType, PlainTextContentType):
				b, _ := io.ReadAll(p.Body)
				log.Ctx(ctx).Info().Str("text", string(b)).Msg("txt body")
			case strings.Contains(contentType, HTMLContentType):
				b, _ := io.ReadAll(p.Body)
				log.Ctx(ctx).Info().Str("html", string(b)).Msg("html body")
			default:
				log.Ctx(ctx).Warn().Str("content_type", contentType).Msg("Inline attachment")
			}
		}
	}
}
