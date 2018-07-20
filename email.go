package main

import (
	"net/smtp"
	"bytes"
	"html/template"
)

// Struct to hold the information for a email message.
type EmailRequest struct {
	from    string
	to      []string
	subject string
	body    string
	server  string
}

func NewEmailRequest(to []string, from string, subject, server, body string) *EmailRequest {
	return &EmailRequest{
		to:      to,
		from:    from,
		subject: subject,
		body:    body,
		server:  server,
	}
}

// Send an email based on the EmailRequest.
func (r *EmailRequest) SendEmail() error {
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	subject := "Subject: " + r.subject + "\n"
	addr := r.server
	emailFrom := r.from

	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()
	c.Mail(emailFrom)
	for _, recipient := range r.to {
		c.Rcpt(recipient)
	}
	wc, err := c.Data()
	if err != nil {
		return err
	}
	defer wc.Close()
	buf := bytes.NewBufferString(subject + mime + "\n" + r.body)
	if _, err = buf.WriteTo(wc); err != nil {
		return err
	}

	return nil
}

// Parse and execute the html/template. There is a function called "zebra" that will handle the
// alternate row colors for each stat line. This stores the output of the template execution into the
// EmailRequest body field.
func (r *EmailRequest) ParseTemplate(templateFileName string, data interface{}) error {
	t := template.New("")
	t.Funcs(template.FuncMap{"zebra": func(i int) bool { return i%2 == 0 }})
	t.Parse(templateFileName)
	buf := new(bytes.Buffer)
	if err := t.Execute(buf, data); err != nil {
		return err
	}
	r.body = buf.String()
	return nil
}

// Run the email summary. This will get the latest stats, then execute the template and finally send
// the email.
func DoEmailSummary(env *Env) error {
	stats := env.db.GetAllToonLatestQuickSummary()

	// The email template laying out the HTML email.
	const tpl = `
<table border="0" cellspacing="0" cellpadding="5">
        <caption>WoW Stats</caption>
    <thead>
    <tr><th>Name</th><th>Level</th><th>Item Level</th><th>Last Modified</th><th>Last Recorded Date</th></tr>
    </thead>
    <tbody>
{{range $idx, $b := .}}
{{if zebra $idx}}<tr bgcolor="#C4C2C2">{{else}}<tr bgcolor="#DBDBDB">{{end}}
<td>{{$b.Toon.Name}}</td><td>{{$b.Level}}</td><td>{{$b.ItemLevel}}</td><td>{{$b.LastModifiedAsDateTime}}</td><td>{{$b.CreateDate.Format "2006-01-02"}}</td></tr>
{{end}}
</tbody></table><p>
`
	r := NewEmailRequest(env.config.Email.ToAddress, env.config.Email.FromAddress, "WoW Stats", env.config.Email.Server, "")
	err := r.ParseTemplate(tpl, stats)
	if err != nil {
		return err
	}

	err = r.SendEmail()
	if err != nil {
		return err
	}
	return nil
}
