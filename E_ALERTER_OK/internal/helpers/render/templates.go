package render

import (
	"bytes"
	"embed"
	"strings"
	"text/template"
	"time"
	"fmt"

	 "middleware/internal/models"
)


var tplFS embed.FS



type payload struct {
	Title    string
	StartFmt string
	EndFmt   string
	Location string
	Changes  []string
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
	
	start, _ := time.Parse(time.RFC3339, evt.Start)
	end,   _ := time.Parse(time.RFC3339, evt.End)

	data := payload{
	Title:    evt.Title,
	StartFmt: start.Format("Monday 02 January 2006 à 15h04"),
	EndFmt:   end.Format("15h04"),
	Location: evt.Location,
	Changes:  humanChanges(evt.Changes),
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

func humanChanges(changes []models.Change) []string {
	var res []string

	for _, c := range changes {
		switch c.Kind {

		case "room_change":
			res = append(res,
				fmt.Sprintf(
					"Le lieu a été modifié : ancienne salle %s, nouvelle salle %s.",
					c.Old, c.New,
				))

		case "time_change":
			res = append(res,
				fmt.Sprintf(
					"L’horaire a été modifié : %s → %s.",
					c.Old, c.New,
				))

		case "title_change":
			res = append(res,
				fmt.Sprintf(
					"Le titre du cours a changé : « %s » → « %s ».",
					c.Old, c.New,
				))

		case "description_change":
			res = append(res,
				"La description du cours a été mise à jour.")

		default:
			res = append(res,
				fmt.Sprintf("%s : %s → %s.", c.Field, c.Old, c.New))
		}
	}
	return res
}
