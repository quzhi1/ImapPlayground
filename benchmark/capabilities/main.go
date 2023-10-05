package main

import (
	"context"
	"crypto/tls"
	"os"

	"github.com/emersion/go-imap/client"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	username = os.Getenv("ICLOUD_EMAIL_ADDRESS")
	password = os.Getenv("ICLOUD_APP_PASSWORD")
)

const (
	// username    = "nylas.sdet@icloud.com"
	// password    = "assu-ndrx-lpzq-xjjg"
	imapAddress = "imap.mail.me.com:993"
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

	// Check capabilities
	caps, err := imapClient.Capability()
	for capability, isSupported := range caps {
		log.Ctx(ctx).Debug().Msgf("Capability %s: %v", capability, isSupported)
	}
}
