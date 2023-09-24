package main

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"
	"strconv"
	"strings"
)

const authTokenKey = "HG_AUTH_TOKEN"
const smtpHostKey = "HG_SMTP_HOST"
const smtpPortKey = "HG_SMTP_PORT"
const smtpUsernameKey = "HG_SMTP_USERNAME"
const smtpPasswordKey = "HG_SMTP_PASSWORD"
const smtpIdentityKey = "HG_SMTP_IDENTITY"
const smtpFromKey = "HG_SMTP_FROM"

type AppSmtpClient struct {
	authToken    string
	smtpHost     string
	smtpPort     uint16
	smtpUsername string
	smtpPassword string
	smtpIdentity string
	smtpFrom     string
}

func (c *AppSmtpClient) InitSmtpClient() error {
	exist := false
	const errTemplate = "unable to read environment variable %s"
	c.authToken, exist = os.LookupEnv(authTokenKey)
	if !exist {
		return fmt.Errorf(errTemplate, authTokenKey)
	}
	c.smtpHost, exist = os.LookupEnv(smtpHostKey)
	if !exist {
		return fmt.Errorf(errTemplate, smtpHostKey)
	}
	smtpPortString, exist := os.LookupEnv(smtpPortKey)
	if !exist {
		return fmt.Errorf(errTemplate, smtpPortKey)
	}
	smtpPort, err := strconv.ParseUint(smtpPortString, 10, 16)
	if err != nil {
		return fmt.Errorf("failed to convert %s=\"%s\" to uint16: %w", smtpPortKey, smtpPortString, err)
	}
	c.smtpPort = uint16(smtpPort)
	c.smtpUsername, exist = os.LookupEnv(smtpUsernameKey)
	if !exist {
		return fmt.Errorf(errTemplate, smtpUsernameKey)
	}
	c.smtpPassword, exist = os.LookupEnv(smtpPasswordKey)
	if !exist {
		return fmt.Errorf(errTemplate, smtpPasswordKey)
	}
	c.smtpIdentity, exist = os.LookupEnv(smtpIdentityKey)
	if !exist {
		return fmt.Errorf(errTemplate, smtpIdentityKey)
	}
	c.smtpFrom, exist = os.LookupEnv(smtpFromKey)
	if !exist {
		return fmt.Errorf(errTemplate, smtpFromKey)
	}
	return nil
}

func (c *AppSmtpClient) SendMail(receiver string, subject string, content string) error {
	headers := map[string]string{
		"From":    c.smtpFrom,
		"To":      receiver,
		"Subject": subject,
	}

	var messageBuilder strings.Builder
	for k, v := range headers {
		messageBuilder.WriteString(k)
		messageBuilder.WriteString(": ")
		messageBuilder.WriteString(v)
		messageBuilder.WriteString("\r\n")
	}
	messageBuilder.WriteString("\r\n")
	messageBuilder.WriteString(content)
	messageBuilder.WriteString("\r\n")

	message := messageBuilder.String()

	tlsConfig := tls.Config{
		ServerName: c.smtpHost,
	}
	hostAddr := fmt.Sprintf("%s:%d", c.smtpHost, c.smtpPort)
	conn, err := tls.Dial("tcp", hostAddr, &tlsConfig)
	defer conn.Close()
	if err != nil {
		return fmt.Errorf("failed to establish secure connection to %s: %w", hostAddr, err)
	}

	smtpClient, err := smtp.NewClient(conn, c.smtpHost)
	defer smtpClient.Close()
	auth := smtp.PlainAuth(c.smtpIdentity, c.smtpUsername, c.smtpPassword, c.smtpHost)

	if err = smtpClient.Auth(auth); err != nil {
		return fmt.Errorf("failed to authenticate with %s as %s: %w", c.smtpHost, c.smtpUsername, err)
	}

	if err = smtpClient.Mail(c.smtpUsername); err != nil {
		return fmt.Errorf("failed to indicate mail from %s: %w", c.smtpFrom, err)
	}

	if err = smtpClient.Rcpt(receiver); err != nil {
		return fmt.Errorf("failed to indicate mail to %s: %w", receiver, err)
	}

	writer, err := smtpClient.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}
	defer writer.Close()

	_, err = writer.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("failed to write message data")
	}

	smtpClient.Quit()
	return nil
}

func main() {

}
