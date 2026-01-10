package render

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"text/template"
	"time"

	"middleware/internal/models"
)

//go:embed templates/*.txt
var tplFS embed.FS

type payload struct {
	Title    string
	StartFmt string
	EndFmt   string
	Location string
	Changes  []string
}

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
	case "event_new":
		path = "templates/event_new.txt"
	default:
		return "[EDT] Notification", evt.EmailText, nil
	}

	raw, err := tplFS.ReadFile(path)
	if err != nil {
		return "[EDT] Notification", evt.EmailText, nil
	}

	subjectTpl, content := splitFrontMatter(raw)
	if strings.TrimSpace(subjectTpl) == "" {
		subjectTpl = "[EDT] Notification"
	}

	start, _ := time.Parse(time.RFC3339, evt.Start)
	end, _ := time.Parse(time.RFC3339, evt.End)

	
	data := payload{
	Title:    evt.Title,
	StartFmt: formatFR(start), 
	EndFmt:   hm(end),        
	Location: evt.Location,
	Changes:  humanChanges(evt.Changes),
		}

	tBody, err := template.New("body").
		Option("missingkey=error").
		Parse(string(content)) 
	if err != nil {
		return "[EDT] Notification", evt.EmailText, nil
	}

	var bodyBuf bytes.Buffer
	if err := tBody.Execute(&bodyBuf, data); err != nil { 
		return "[EDT] Notification", evt.EmailText, nil
	}

	tSubj, err := template.New("subject").
		Option("missingkey=error").
		Parse(subjectTpl) 
	if err != nil {
		return "[EDT] Notification", evt.EmailText, nil
	}

	var subjBuf bytes.Buffer
	if err := tSubj.Execute(&subjBuf, data); err != nil {
		return "[EDT] Notification", evt.EmailText, nil
	}

	return strings.TrimSpace(subjBuf.String()), bodyBuf.String(), nil 
}

func humanChanges(changes []models.Change) []string {
	var res []string

	for _, c := range changes {
		switch c.Kind {
		case "room_change":
			res = append(res,
				fmt.Sprintf("Le lieu a été modifié : ancienne salle %s, nouvelle salle %s.", c.Old, c.New))
		/*case "time_change":
			res = append(res,
				fmt.Sprintf("L’horaire a été modifié : %s → %s.", c.Old, c.New))*/
				
		case "time_change":
		res = append(res,
        		fmt.Sprintf("L’horaire a été modifié : %s → %s.", fmtRFC3339FR(c.Old), fmtRFC3339FR(c.New)))
		
		case "title_change":
			res = append(res,
				fmt.Sprintf("Le titre du cours a changé : « %s » → « %s ».", c.Old, c.New))
		case "description_change":
			res = append(res, "La description du cours a été mise à jour.")
		default:
			res = append(res, fmt.Sprintf("%s : %s → %s.", c.Field, c.Old, c.New))
		}
	}
	return res
}


func formatFR(t time.Time) string {
	jours := []string{"dimanche", "lundi", "mardi", "mercredi", "jeudi", "vendredi", "samedi"}
	mois := []string{"janvier", "février", "mars", "avril", "mai", "juin", "juillet", "août", "septembre", "octobre", "novembre", "décembre"}

	return fmt.Sprintf("%s %02d %s %d à %02dh%02d",
		jours[int(t.Weekday())],
		t.Day(),
		mois[int(t.Month())-1],
		t.Year(),
		t.Hour(),
		t.Minute(),
	)
}

func hm(t time.Time) string {
	return fmt.Sprintf("%02dh%02d", t.Hour(), t.Minute())
}

func fmtRFC3339FR(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s // fallback si c’est pas du RFC3339
	}
	return formatFR(t)
}

func fmtRFC3339HM(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	return hm(t)
}






