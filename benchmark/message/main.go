package main

import (
	"context"
	"crypto/tls"
	"io"
	"os"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	imapAddress = "imap.mail.yahoo.com:993"
	folderName  = "Inbox"
	syncPeriod  = 30 * 24 * time.Hour
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

	// Read auth
	username := os.Getenv("YAHOO_EMAIL_ADDRESS")
	password := os.Getenv("YAHOO_APP_PASSWORD")

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

	// Select folder
	folder, err := imapClient.Select(folderName, false)
	if err != nil {
		panic(err)
	}
	log.Ctx(ctx).Debug().Str("folderName", folderName).Uint32("UIDVALIDITY", folder.UidValidity).Msg("Selected folder")

	// Get message UIDs
	log.Ctx(ctx).Debug().Float64("syncPeriod", syncPeriod.Seconds()).Msg("Get message UIDs")
	criteria := imap.NewSearchCriteria()
	criteria.Since = time.Now().Add(-syncPeriod)
	uids, err := imapClient.UidSearch(criteria)
	if err != nil {
		panic(err)
	}

	// Get messages
	log.Ctx(ctx).Debug().Msgf("Fetching messages: %v", uids)
	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)
	var section imap.BodySectionName
	items := []imap.FetchItem{section.FetchItem()}
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
		}

		r := msg.GetBody(&section)
		if r == nil {
			log.Ctx(ctx).Fatal().Msg("Server didn't returned message body")
		}

		// Create a new mail reader
		mr, err := mail.CreateReader(r)
		if err != nil {
			panic(err)
		}

		// Print some info about the message
		header := mr.Header
		if date, err := header.Date(); err == nil {
			log.Ctx(ctx).Info().Msgf("Date: %s", date)
		}
		if from, err := header.AddressList("From"); err == nil {
			log.Ctx(ctx).Info().Msgf("From: %s", from)
		}
		if to, err := header.AddressList("To"); err == nil {
			log.Ctx(ctx).Info().Msgf("To: %s", to)
		}
		if subject, err := header.Subject(); err == nil {
			log.Ctx(ctx).Info().Msgf("Subject: %s", subject)
		}

		// Process each message's part
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			} else if err != nil {
				panic(err)
			}

			switch h := p.Header.(type) {
			case *mail.InlineHeader:
				// This is the message's text (can be plain-text or HTML)
				b, _ := io.ReadAll(p.Body)
				log.Ctx(ctx).Info().Msgf("Got text: %v", string(b))
			case *mail.AttachmentHeader:
				// This is an attachment
				filename, _ := h.Filename()
				log.Ctx(ctx).Info().Msgf("Got attachment: %v", filename)

				// Print attachment
				buf := new(strings.Builder)
				_, err := io.Copy(buf, p.Body)
				if err != nil {
					panic(err)
				}
				log.Ctx(ctx).Info().Msgf("Attachment: %s", buf.String())
			}
		}
	}

	// Logout
	err = imapClient.Logout()
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Error logging out of IMAP server. We will directly close the connection")
	}
}
