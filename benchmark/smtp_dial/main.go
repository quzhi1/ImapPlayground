package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"

	"github.com/rs/zerolog/log"
)

const (
	HandshakeTLSPort = 587
	AutoTLSPort      = 465
)

type smtpAddress struct {
	host string
	port int
}

func main() {
	// Create a context with logger
	ctx := log.Logger.WithContext(context.Background())

	// SMTP server configuration.
	smtpAddresses := []smtpAddress{
		{
			host: "smtp.example.com",
			port: 25, // Invalid
		},
		{
			host: "smtp.mail.yahoo.com",
			port: 587, // Handshake TLS
		},
		{
			host: "smtp.mail.yahoo.com",
			port: 465, // Implicit TLS
		},
	}

	// Connect to the SMTP server.
	for _, smtpAddress := range smtpAddresses {
		err := connect(ctx, smtpAddress.host, smtpAddress.port)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).
				Str("host", smtpAddress.host).
				Int("port", smtpAddress.port).
				Msg("failed validation")
		} else {
			log.Ctx(ctx).Info().Str("host", smtpAddress.host).Int("port", smtpAddress.port).Msg("validation successful")
		}
	}
}

func connect(ctx context.Context, host string, port int) error {
	if port == HandshakeTLSPort {
		return dialWithTLSHandshake(ctx, host, port)
	} else {
		config := &tls.Config{
			ServerName:         host,
			InsecureSkipVerify: true,
		}
		conn, err := tls.Dial("tcp", fmt.Sprintf("%v:%v", host, port), config)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msgf("implicit TLS failed, trying handshake TLS")
			return dialWithTLSHandshake(ctx, host, port)
		} else {
			defer conn.Close()
		}
		return nil
	}
}

func dialWithTLSHandshake(ctx context.Context, host string, port int) error {
	client, err := smtp.Dial(fmt.Sprintf("%v:%v", host, port))
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("dial with TLS handshake failed")
		return err
	} else {
		defer client.Close()
		return nil
	}
}
