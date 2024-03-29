package main

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	contentTypeRegex = regexp.MustCompile(".*name=\"(\\S+)\"")
	username         = os.Getenv("ICLOUD_EMAIL_ADDRESS_MANY_MESSAGES")
	password         = os.Getenv("ICLOUD_APP_PASSWORD_MANY_MESSAGES")
)

const (
	// imapAddress = "imap.mail.yahoo.com:993"
	imapAddress          = "imap.mail.me.com:993"
	folderName           = "INBOX"
	HTMLContentType      = "text/html"
	PlainTextContentType = "text/plain"
	multipartError       = "multipart: NextPart: EOF"
	uid                  = 302
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

	// Select folder
	folder, err := imapClient.Select(folderName, true)
	if err != nil {
		panic(err)
	}
	log.Ctx(ctx).Debug().Str("folderName", folderName).Uint32("UIDVALIDITY", folder.UidValidity).Msg("Selected folder")

	// Get messages
	log.Ctx(ctx).Debug().Msgf("Fetching message: %v", uid)
	seqset := new(imap.SeqSet)
	seqset.AddNum(uid)
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
			part, err := mr.NextPart()
			if errors.Is(err, io.EOF) {
				break
			} else if err != nil {
				shouldBreak := false
				switch {
				case message.IsUnknownCharset(err):
					log.Ctx(ctx).Warn().Err(err).Msg("Ignore unknown charset")
				case strings.Contains(err.Error(), multipartError):
					log.Ctx(ctx).Warn().Err(err).Msg("Ignore multipart error")
					shouldBreak = true
				default:
					log.Ctx(ctx).Error().Err(err).Msg("Error reading message part")
					panic(err)
				}

				if shouldBreak {
					break
				}
			}

			switch partHeader := part.Header.(type) {
			case *mail.InlineHeader:
				contentType := partHeader.Header.Header.Get("Content-Type")
				switch {
				case strings.Contains(contentType, PlainTextContentType):
					textBody := readerToString(ctx, part.Body)
					log.Ctx(ctx).Debug().Str("text", textBody).Msg("Found text body")
				case strings.Contains(contentType, HTMLContentType):
					htmlBody := readerToString(ctx, part.Body)
					log.Ctx(ctx).Debug().Str("html", htmlBody).Msg("Found html body")
				default:
					printAttachment(ctx, partHeader.Header, part)
				}
			case *mail.AttachmentHeader:
				printAttachment(ctx, partHeader.Header, part)
			}
		}
	}

	// Logout
	err = imapClient.Logout()
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Error logging out of IMAP server. We will directly close the connection")
	}
}

func printAttachment(ctx context.Context, header message.Header, part *mail.Part) {
	contentType := header.Header.Get("Content-Type")

	// Get or generate file ID
	var fileID string
	if header.Header.Has("X-Attachment-Id") {
		fileID = header.Header.Get("X-Attachment-Id")
	} else {
		fileID = uuid.New().String()
		log.Ctx(ctx).Warn().
			Str("content_type", contentType).
			Str("file_id", fileID).
			Msg("Attachment does not have attachment id. Probably in draft folder. Generating one")
	}

	// Read file name
	var fileName string
	match := contentTypeRegex.FindStringSubmatch(contentType)
	if match != nil {
		// Extract the captured string from the first submatch
		fileName = match[1]
	} else {
		log.Ctx(ctx).Warn().
			Str("content_type", contentType).
			Str("file_id", fileID).
			Msg("Attachment has file name in Content-Type header, setting it to empty")
	}

	// Read content disposition
	var contentDisposition string
	if header.Header.Has("Content-Disposition") {
		contentDisposition = header.Header.Get("Content-Disposition")
	} else {
		log.Ctx(ctx).Warn().
			Str("content_type", contentType).
			Str("file_id", fileID).
			Msg("Attachment has no Content-Disposition header, setting it to empty")
	}

	// Read file content
	log.Ctx(ctx).Debug().
		Str("content_type", contentType).
		Str("file_id", fileID).
		Str("file_name", fileName).
		Str("content_disposition", contentDisposition).
		Msg("Found attachment")

	// Write attachment
	err := writeToFile(ctx, "tmp.txt", part.Body)
	if err != nil {
		panic(err)
	}
}

func readerToString(ctx context.Context, reader io.Reader) string {
	buf := new(strings.Builder)
	_, err := io.Copy(buf, reader)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to read file content, setting it to empty")

	}
	return buf.String()
}

func writeToFile(ctx context.Context, filename string, reader io.Reader) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	if err != nil {
		if !strings.Contains(err.Error(), io.EOF.Error()) {
			return err
		}
	}

	return nil
}
