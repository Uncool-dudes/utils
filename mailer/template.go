package mailer

import (
	"bytes"
	htmltemplate "html/template"
	texttemplate "text/template"
)

// RenderText executes tmpl with data and returns a text/plain Part.
func RenderText(tmpl *texttemplate.Template, data any) (Part, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return Part{}, Domain.Wrap(err, "render text template")
	}
	return TextPart(buf.String()), nil
}

// RenderHTML executes tmpl with data and returns a text/html Part.
func RenderHTML(tmpl *htmltemplate.Template, data any) (Part, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return Part{}, Domain.Wrap(err, "render html template")
	}
	return HTMLPart(buf.String()), nil
}
