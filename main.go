package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
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
const listenAddrKey = "HG_LISTEN_ADDR"

const smtpFrom = "hackergame@ustclug.org"

type AppSmtpClient struct {
	authToken    string
	smtpHost     string
	smtpPort     uint16
	smtpUsername string
	smtpPassword string
	smtpIdentity string
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
	return nil
}

func (c *AppSmtpClient) SendMail(receiver string, subject string, content string) error {
	headers := map[string]string{
		"From":    smtpFrom,
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

	if err = smtpClient.Mail(c.smtpIdentity); err != nil {
		return fmt.Errorf("failed to indicate mail from %s: %w", c.smtpIdentity, err)
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

type HttpError struct {
	status int
	err    error
}

func NewHttpError(status int, err error) *HttpError {
	return &HttpError{
		status: status,
		err:    err,
	}
}

type MailApiResponse struct {
	Success bool   `json:"success"`
	Message string `json:"msg"`
}

type MailApiRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	IP      string `json:"ip"`
}

func (h *HttpError) Error() string {
	return h.err.Error()
}

func (c *AppSmtpClient) MailApiHandler(w http.ResponseWriter, r *http.Request) error {
	token := r.Header.Get("Authorization")
	if len(token) < 7 || !strings.HasPrefix(token, "Bearer") {
		return NewHttpError(http.StatusUnauthorized, fmt.Errorf("Malformed Token"))
	}
	token = token[7:]
	if token != c.authToken {
		return NewHttpError(http.StatusUnauthorized, fmt.Errorf("Malformed Token"))
	}
	request := MailApiRequest{}
	reqBuffer := make([]byte, 4096)
	reqReader := r.Body
	reqLen, err := reqReader.Read(reqBuffer)
	if err != nil && !errors.Is(err, io.EOF) {
		return NewHttpError(http.StatusInternalServerError, fmt.Errorf("failed to read request: %w", err))
	}
	reqBuffer = reqBuffer[:reqLen]
	err = json.Unmarshal(reqBuffer, &request)
	if err != nil {
		return NewHttpError(http.StatusBadRequest, fmt.Errorf("failed to parse request: %w", err))
	}
	err = c.SendMail(request.To, request.Subject, request.Body)
	if err != nil {
		log.Printf("Failed to send mail to %s: %w", request.To, err)
		return NewHttpError(http.StatusInternalServerError, fmt.Errorf("failed to send mail to %s: %w", request.To, err))
	}
	log.Printf("Successfully send mail to %s, from %s", request.To, request.IP)
	return nil
}

func (c *AppSmtpClient) MailApiHandlerWrapper(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	err := c.MailApiHandler(w, r)
	var resp []byte
	if err != nil {
		resp, err = json.Marshal(MailApiResponse{
			Success: false,
			Message: err.Error(),
		})
	} else {
		resp, err = json.Marshal(MailApiResponse{
			Success: true,
			Message: "",
		})
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte(fmt.Errorf("failed to serialize repsonse: %w", err).Error()))
	} else {
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(resp)
	}
	if err != nil {
		log.Printf("Failed to write response: %w", err)
	}
}

func main() {
	app := AppSmtpClient{}
	err := app.InitSmtpClient()
	if err != nil {
		log.Fatalf("Failed to initialize: %s", err)
	}
	listenAddr, exist := os.LookupEnv(listenAddrKey)
	if !exist {
		listenAddr = ":8080"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/mail", app.MailApiHandlerWrapper)
	log.Fatal(http.ListenAndServe(listenAddr, mux))
}
