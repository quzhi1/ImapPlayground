# ImapPlayground

## Prerequisite
1. Create an Yahoo App password: https://login.yahoo.com/myaccount/security/?scrumb=P5NQao1naWv&anchorId=appPasswordCard
2. Store password as env var (in your .zshrc):
```bash
# Yahoo App password
export YAHOO_EMAIL_ADDRESS=<your-email-address>
export YAHOO_APP_PASSWORD=<your-app-password>
```

## Talk to Yahoo IMAP server
```bash
openssl s_client -connect imap.mail.yahoo.com:993 -crlf -quiet
# Command line will hang
tag login <your-email-address> <your-app-password>
```

## Create a user in local IMAP
```bash
curl -L -X POST 'localhost:8080/api/user' \
-H 'Content-Type: application/json' \
--data-raw '{
  "email": "test@localhost",
  "login": "test",
  "password": "test"
}'
```

## Talk to local IMAP server
```bash
openssl s_client -connect localhost:3993 -crlf -quiet
# Command line will hang
tag login test@localhost test
```

## Some example command
```bash
# See available commands
tag CAPABILITY
# List folders
tag LIST "" "*"
# Select folders
tag SELECT INBOX
tag SELECT Inbox
tag SELECT Archive
# Count messages in folder
tag STATUS INBOX (MESSAGES)
tag STATUS Archive (MESSAGES)
# Check last 10 messages
tag FETCH 10:10 (BODY[HEADER])
# Check the 10th message
tag FETCH 10 (BODY)
# List MIME for 10th message
tag FETCH 10 (BODY[HEADER])
# Check the 10th message, part 0
tag FETCH 6388 (BODY[0])
# Log out
tag LOGOUT
# IDLE
tag IDLE
```

## Run proof of concept
```bash
go run client/main.go
```