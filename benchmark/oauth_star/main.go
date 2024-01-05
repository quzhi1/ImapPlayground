package main

import (
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"log"
)

const (
	server      = "imap.mail.yahoo.com:993"
	email       = "your_email@yahoo.com"
	accessToken = "bOHK0bufuFvfNlNG1ZvhWUCs_8Fs8xM.d2OuiPSni6YrA8_0xllRp9_xeA8P1MVF.hEOAJGML3Iuv6vy5kEq.bk_lpeIUCUvg2mIzUj2ejyjrmHFHRqWYBkHcWemcbojZCZ2uxyTYEp70NaYEjmaTrhlgIhY8J3Y5kEP_k_sE7xS79wHGr46NgfGGu.wqsccLdQgcxk5zvC__9n0iT3ZAi92.3XYL6rtzR9o21u9eF0dmsY.ONkzKAQvH2eik92MLYA.KAUfhlTUWYY5TlZa97pnE82eCX4Qv7FX1lPQN95s.Loqb0T7xm1hKQIJ35ws_9xBvInpG6AzN2kSe0.7jerEboz35bGPXX8cz5h5HnDKpTNY2nyuLQd7aYTZw9Za4NAtgSuIKS4JFnzueGMGYomDccHpeS36IYp_3Bgv2rZukhHv1K0c9_49YIPD6vw.UgY974uS0.WxEKng9sqhgZ761N7_coDiFIWd7o3ypkTJ3M6u5yWZXsggkIhq.WEYlEH8v437z.S.V1iPEy5_YX.zUT.8mvzeD__bwiDSYUMxb.XykHBmngc3CmE1uRb6WtiUorsA83CSn4JonQ0aKzQ.WWbM3moPysRcpKDZXpZHcR23hNyHqUPQDQdhhgT7eNmX7oVlJAOBV0JzmEqX.zLv45YTn0QjEtjU7xLR2bzuKJQ2CIO5yMNlHnla1Tcu9ScJ9PUJFtt8VKRNvAJjbq1x15Ol42eiIoOOuUChMBCGmUL79WfF0SLhuXB12NZE7eUwy_DGQvCCc9W8.KMpfbhZ7ai9Yx_Hw5lTL57yfKFBt2ut9xZzr2zzddhwNyNt_FpHNVOL6sJQyTOlW6xUZB0H04ImNWLA42HwQdFczIPonhRFT9ASjF_WYXMNbbCnwwRUTfs_7iepe996amlf73NJ5kOQCVWoZZKCSKjKOuK0QptR6Cdrb0ZFST0V6LTfdoEWSrBcNj3OSlJo2nQezxW2Pci_RAMPqoXF2TUP1AliwP4pwPMgP_Pnljg-"
)

func main() {

	// Connect to the server
	c, err := client.DialTLS(server, nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected")

	// Authenticate
	auth := oauth2.NewXOAuth2(email, accessToken)
	if err := c.Authenticate(auth); err != nil {
		log.Fatal(err)
	}
	log.Println("Authenticated")

	// Select INBOX
	_, err = c.Select("INBOX", false)
	if err != nil {
		log.Fatal(err)
	}

	// Search for a specific message
	// Here you need to define how you want to search for your message. This is just an example.
	criteria := imap.NewSearchCriteria()
	criteria.Header.Add("Subject", "Subject of the email to star")
	uids, err := c.Search(criteria)
	if err != nil {
		log.Fatal(err)
	}

	// Assuming the message exists and its UID is known
	if len(uids) > 0 {
		seqSet := new(imap.SeqSet)
		seqSet.AddNum(uids...)

		// Set the flag
		item := imap.FormatFlagsOp(imap.AddFlags, true)
		flags := []interface{}{imap.Flagged}
		err = c.Store(seqSet, item, flags, nil)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Message starred")
	} else {
		log.Println("No messages found")
	}

	// Logout
	if err := c.Logout(); err != nil {
		log.Fatal(err)
	}
}
