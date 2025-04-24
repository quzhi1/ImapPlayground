package main

import (
	"context"
	"crypto/tls"
	"os"
	"reflect"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	username = os.Getenv("SSI_EMAIL_ADDRESS")
	password = os.Getenv("SSI_PASSWORD")
)

const (
	imapAddress          = "email.sscihosting.com:993"
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
	// Default method.
	log.Ctx(ctx).Debug().Str("imapAddress", imapAddress).Msg("Connecting to IMAP server")
	imapClient, err := imapclient.DialTLS(imapAddress, &imapclient.Options{
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // We support self signed imap server
			CipherSuites:       allCipherSuites(),
		},
		DebugWriter: os.Stderr,
	})
	// Fallback to StartTLS command method.
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("Failed to connect to IMAP server using TLS. Fallback to StartTLS")
		imapClient, err = imapclient.DialStartTLS(imapAddress, &imapclient.Options{
			TLSConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // We support self signed imap server
				CipherSuites:       allCipherSuites(),
			},
		})
	}
	// Fallback to insecure method.
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("Falling back to insecure connection")
		imapClient, err = imapclient.DialInsecure(imapAddress, &imapclient.Options{
			TLSConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // We support self signed imap server
				CipherSuites:       allCipherSuites(),
			},
		})
		if err != nil {
			panic(err)
		}
	}

	// Verify connection works
	capSet, err := imapClient.Capability().Wait()
	if err != nil {
		panic(err)
	}
	log.Ctx(ctx).Debug().Any("capabilities", capSet).Msg("IMAP server capabilities")

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
			panic("error selecting " + folder.Mailbox + " " + err.Error())
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

func searchOneFolder(ctx context.Context, imapClient *imapclient.Client, folderName string) []imap.UID {
	// Search for messages in the last 90 days
	criteria := imap.SearchCriteria{
		SentSince: time.Now().AddDate(0, 0, -90),
	}
	log.Ctx(ctx).Debug().Str("folderName", folderName).Msg("Searching folder")
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
				log.Ctx(ctx).Debug().
					Str("message_id", item.Envelope.MessageID).
					Str("subject", item.Envelope.Subject).
					Msg("Reading message ID")
			case imapclient.FetchItemDataBodySection:
				// b, err := io.ReadAll(item.Literal)
				// if err != nil {
				// 	panic(err)
				// }
				// fmt.Println(string(b))
			case imapclient.FetchItemDataFlags:
				// log.Ctx(ctx).Debug().Any("flags", item.Flags).Msg("Reading flags")
			case imapclient.FetchItemDataUID:
				// log.Ctx(ctx).Debug().Uint32("uid", uint32(item.UID)).Msg("Reading UID")
			default:
				log.Ctx(ctx).Warn().Str("fetch_item_type", reflect.TypeOf(item).String()).Msg("Unknown fetch item type")
			}
		}
	}
}

// allCipherSuites returns all ciphers supported by the Go standard library.
// It includes both secure and insecure ciphers.
func allCipherSuites() []uint16 {
	var uint16CipherSuites = []uint16{}
	for _, suite := range tls.CipherSuites() {
		uint16CipherSuites = append(uint16CipherSuites, suite.ID)
	}
	for _, suite := range tls.InsecureCipherSuites() {
		uint16CipherSuites = append(uint16CipherSuites, suite.ID)
	}
	return uint16CipherSuites
}
