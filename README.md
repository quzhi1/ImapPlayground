# ImapPlayground

## Prerequisite
1. Create an Yahoo App password: https://login.yahoo.com/myaccount/security/?scrumb=P5NQao1naWv&anchorId=appPasswordCard
2. Store password as env var (in your .zshrc):
```bash
# Yahoo App password
export YAHOO_EMAIL_ADDRESS=<your-email-address>
export YAHOO_APP_PASSWORD=<your-app-password>
```

## Connect and log into Yahoo IMAP server
```bash
openssl s_client -connect imap.mail.yahoo.com:993 -crlf -quiet
# Command line will hang
a login <your-email-address> <your-app-password>
```

## Connect and log into other IMAP servers
```bash
openssl s_client -connect <imap-host>:<imap-port> -starttls imap -crlf -quiet -ign_eof # With startTLS
openssl s_client -connect <imap-host>:<imap-port> -crlf -quiet # No startTLS
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

## Connect and log into local IMAP server
```bash
openssl s_client -connect localhost:3993 -crlf -quiet

# Command line will hang
a login test@localhost test
a login user-a@localhost user-a
```

## Some example command
```bash
# See available commands
a CAPABILITY
# List folders
a LIST "" "*"
a LIST "" "%"
# Select folders
a SELECT INBOX
a SELECT Inbox
a SELECT Archive
# Count messages in folder
a STATUS INBOX (MESSAGES)
a STATUS Archive (MESSAGES)
# List all UIDs
a uid search all
a uid search SENTSINCE 01-Mar-2024
a uid search NOT TEXT Zhi
# Change flags
a STORE 1 +FLAGS (\abc)
# Check last 10 messages
a FETCH 10:10 (BODY[HEADER])
# Check the 10th message
a FETCH 10 (BODY)
# List MIME for 10th message
a FETCH 10 (BODY[HEADER])
# Check the 10th message, part 0
a FETCH 6388 (BODY[0])
# Log out
a LOGOUT
# IDLE
a IDLE
# ID
a id ("name" "Yahoo Mail Client" "version" "1.0" "os" "Linux" "os-version" "7.9.4" "vendor" "Yahoo")
```

## Run proof of concept
```bash
go run client/main.go
```