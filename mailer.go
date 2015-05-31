package main

import (
	"bytes"
	"fmt"
	"github.com/mailgun/mailgun-go"
	"html/template"
)

type Mailer interface {
	Notify(video *DownloadVideo)
}

type MailgunMailer struct {
	Mailgun         mailgun.Mailgun
	SuccessTemplate *template.Template
	ErrorTemplate   *template.Template
}

func NewMailgunMailer(key string, domain string) (*MailgunMailer, error) {
	mg := &MailgunMailer{}
	mg.Mailgun = mailgun.NewMailgun(domain, key, "")
	var err error
	mg.SuccessTemplate, err = template.New("success").Parse(`
Hola {{.Name}}:

Tu video "{{.Title}}"" está listo, puedes descargarlo desde {{.DstUrl}}.

saludos`)
	if err != nil {
		return nil, err
	}
	mg.ErrorTemplate, err = template.New("error").Parse(`
Hola {{.Name}}:

Hubo un error al descargar el video "{{.Title}}": {{.Error}}

saludos`)
	if err != nil {
		return nil, err
	}
	return mg, nil
}

func (mailer *MailgunMailer) Notify(video *DownloadVideo) {
	var subject string
	txt := bytes.NewBufferString("")

	if video.Error != nil {
		subject = fmt.Sprintf("%s, hubo un error en la descarga de %s :(", video.Name, video.Title)
		mailer.ErrorTemplate.Execute(txt, video)
	} else {
		subject = fmt.Sprintf("%s, tu video %s está listo", video.Name, video.Title)
		mailer.SuccessTemplate.Execute(txt, video)
	}

	msg := mailer.Mailgun.NewMessage("yutubaas@larix.io", subject, txt.String(), video.Email)

	if mes, id, err := mailer.Mailgun.Send(msg); err != nil {
		log.Error("error sending email to mailgun: %s", err)
	} else {
		log.Debug("message sent to mailgun: id=%s status=%s", id, mes)
	}
}
