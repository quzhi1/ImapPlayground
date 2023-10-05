package main

import (
	"context"
	"crypto/tls"
	"github.com/emersion/go-message/mail"
	"github.com/quzhi1/go-imap"
	"github.com/quzhi1/go-imap/client"
	"github.com/quzhi1/go-sasl"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"io"
	"os"
	"strings"
)

const (
	imapAddress = "imap.mail.yahoo.com:993"
	folderName  = "Inbox"
	username    = "ps_21_69@yahoo.com"
	// Fill it with your own access token
	accessToken = ""
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
	if err := imapClient.Authenticate(saslClient); err != nil {
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

	// Get message UIDs
	log.Ctx(ctx).Debug().Msg("Get message UIDs")
	criteria := imap.NewSearchCriteria()
	uids, err := imapClient.UidSearch(criteria)
	if err != nil {
		panic(err)
	}

	// Get messages
	if len(uids) == 0 {
		log.Ctx(ctx).Debug().Msg("No messages to fetch")
		return
	}
	log.Ctx(ctx).Debug().Msgf("Fetching messages: %v", uids)
	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)
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
				// b, _ := io.ReadAll(p.Body)
				// log.Ctx(ctx).Info().Msgf("Got text: %v", string(b))
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
				// log.Ctx(ctx).Info().Msgf("Attachment: %s", buf.String())
			}
		}
	}

	// Logout
	err = imapClient.Logout()
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Error logging out of IMAP server. We will directly close the connection")
	}
}

func listFolder(ctx context.Context, client *client.Client) []imap.MailboxInfo {
	mailboxes := make(chan *imap.MailboxInfo, 100)
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
