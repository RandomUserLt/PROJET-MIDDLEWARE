package render

import (
	"bytes"
	"embed"
	"strings"
	"text/template"

	"middleware/alerter/internal/models"
)

// On embarque tout le répertoire templates/ dans ce package.
//go:embed templates/*
var tplFS embed.FS

type payload struct {
	Title     string
	Start     string
	End       string
	Location  string
	Changes   []models.Change
	EmailText string // fallback
}

// parse un front-matter minimal de la forme:
// ---\n
// subject: "..."\n
// ---\n
// <contenu du template>
func splitFrontMatter(raw []byte) (subject string, content []byte) {
	s := string(raw)
	if !strings.HasPrefix(s, "---\n") {
		return "", raw
	}
	rest := s[len("---\n"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", raw
	}
	header := rest[:end]
	body := rest[end+len("\n---"):]
	subj := ""
	for _, ln := range strings.Split(header, "\n") {
		ln = strings.TrimSpace(ln)
		if strings.HasPrefix(ln, "subject:") {
			subj = strings.TrimSpace(strings.TrimPrefix(ln, "subject:"))
			subj = strings.Trim(subj, `"'`)
			break
		}
	}
	return subj, []byte(body)
}

func RenderMail(evt models.AlertEvent) (subject string, body string, err error) {
	var path string
	switch evt.Type {
	case "event_changed":
		path = "templates/event_changed.txt"
	default:
		path = "templates/event_new.txt"
	}

	raw, err := tplFS.ReadFile(path)
	if err != nil {
		return "", "", err
	}

	subject, content := splitFrontMatter(raw)

	t, err := template.New("mail").Parse(string(content))
	if err != nil {
		return "", "", err
	}

	buf := new(bytes.Buffer)
	data := payload{
		Title:     evt.Title,
		Start:     evt.Start,
		End:       evt.End,
		Location:  evt.Location,
		Changes:   evt.Changes,
		EmailText: evt.EmailText,
	}
	if err := t.Execute(buf, data); err != nil {
		return "", "", err
	}
	//new treatment here
	subjT, err := template.New("subject").Parse(subject)
	if err != nil { return "", "", err }
	var subjBuf bytes.Buffer
	if err := subjT.Execute(&subjBuf, data); err != nil {
	    return "", "", err
	}
	subject = subjBuf.String()

	return subject, buf.String(), nil
	
	
	//return subject, buf.String(), nil
}


