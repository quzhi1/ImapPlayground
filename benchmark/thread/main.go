package main

import (
	"context"
	"crypto/tls"
	"io"
	"os"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	// imapAddress = "imap.mail.yahoo.com:993"
	imapAddress = "imap.mail.me.com:993"
	folderName  = "Sent Messages"
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
	// username := os.Getenv("YAHOO_EMAIL_ADDRESS")
	// password := os.Getenv("YAHOO_APP_PASSWORD")
	username := os.Getenv("ICLOUD_EMAIL_ADDRESS")
	password := os.Getenv("ICLOUD_APP_PASSWORD")

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
	folder, err := imapClient.Select(folderName, true)
	if err != nil {
		panic(err)
	}
	log.Ctx(ctx).Debug().Str("folderName", folderName).Uint32("UIDVALIDITY", folder.UidValidity).Msg("Selected folder")

	// Get messages
	log.Ctx(ctx).Debug().Msg("Fetching third message")
	seqset := new(imap.SeqSet)
	seqset.AddNum(3)
	// Setting peek to true will prevent marking the messages as read
	section := imap.BodySectionName{Peek: true}
	items := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchFlags,
		imap.FetchInternalDate,
		section.FetchItem(),
	}
	messageChans := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- imapClient.Fetch(seqset, items, messageChans)
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

		// Print internal date
		log.Ctx(ctx).Info().Msgf("InternalDate: %s", msg.InternalDate)

		// References
		if mr.Header.Has("References") {
			log.Ctx(ctx).Info().Msgf("References: %v", mr.Header.Get("References"))
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
				log.Ctx(ctx).Info().Msgf("Got text Content-Type: %s", h.Header.Header.Get("Content-Type"))
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
