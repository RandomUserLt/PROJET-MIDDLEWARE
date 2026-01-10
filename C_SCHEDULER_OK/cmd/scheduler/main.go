package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/zhashkevych/scheduler"
)

type Event struct {
	ID          string   `json:"id"`
	AgendaIDs   []string `json:"agendaIds"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Start       string   `json:"start"` // RFC3339
	End         string   `json:"end"`   // RFC3339
	Location    string   `json:"location"`
	LastUpdate  string   `json:"lastUpdate"`
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}


var reIdent = regexp.MustCompile(`Identifiant\s*:\s*([0-9]+)`)

func parseAgendaIDsFromConfigPlainText(s string) []string {
	matches := reIdent.FindAllStringSubmatch(s, -1)
	seen := map[string]bool{}
	var ids []string
	for _, m := range matches {
		id := strings.TrimSpace(m[1])
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

func filterLikelyUCAResources(ids []string) []string {
	var out []string
	for _, id := range ids {
		if len(id) >= 4 {
			out = append(out, id)
		}
	}
	return out
}



func buildIcalURL(icalBase string, ids []string) (string, error) {
	u, err := url.Parse(icalBase)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("resources", strings.Join(ids, ","))
	u.RawQuery = q.Encode()
	return u.String(), nil
}


func unfoldICalLines(raw []byte) []string {
	sc := bufio.NewScanner(bytes.NewReader(raw))
	var lines []string
	var cur string

	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			cur += strings.TrimLeft(line, " \t")
			continue
		}
		if cur != "" {
			lines = append(lines, cur)
		}
		cur = line
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

func parseICalTime(v string, loc *time.Location) (time.Time, error) {
	v = strings.TrimSpace(v)

	layoutsZ := []string{
		"20060102T150405Z",
		"20060102T1504Z",
	}
	layoutsLocal := []string{
		"20060102T150405",
		"20060102T1504",
		"20060102",
	}

	if strings.HasSuffix(v, "Z") {
		for _, layout := range layoutsZ {
			if t, err := time.Parse(layout, v); err == nil {
				return t, nil
			}
		}
	}

	for _, layout := range layoutsLocal {
		if t, err := time.ParseInLocation(layout, v, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported iCal datetime: %q", v)
}

func splitICalLine(line string) (key string, value string) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return "", ""
	}
	left := parts[0]
	value = parts[1]

	key = strings.SplitN(left, ";", 2)[0]
	return strings.ToUpper(strings.TrimSpace(key)), strings.TrimSpace(value)
}

func parseEventsFromICal(raw []byte, agendaID string) ([]Event, error) {
	loc, _ := time.LoadLocation("Europe/Paris")
	lines := unfoldICalLines(raw)

	var events []Event
	inEvent := false
	props := map[string]string{}

	flush := func() {
		uid := props["UID"]
		if uid == "" {
			return
		}

		startRaw := props["DTSTART"]
		endRaw := props["DTEND"]

		startT, err := parseICalTime(startRaw, loc)
		if err != nil {
			return
		}
		endT, err := parseICalTime(endRaw, loc)
		if err != nil {
			return
		}

		var lastUpdate string
		if lm := props["LAST-MODIFIED"]; lm != "" {
			if t, err := parseICalTime(lm, loc); err == nil {
				lastUpdate = t.UTC().Format(time.RFC3339)
			}
		}

		events = append(events, Event{
			ID:          uid + "|" + agendaID,      
			AgendaIDs:   []string{agendaID},      
			Title:       props["SUMMARY"],
			Description: props["DESCRIPTION"],
			Start:       startT.UTC().Format(time.RFC3339),
			End:         endT.UTC().Format(time.RFC3339),
			Location:    props["LOCATION"],
			LastUpdate:  lastUpdate,
		})
	}

	for _, line := range lines {
		switch strings.ToUpper(strings.TrimSpace(line)) {
		case "BEGIN:VEVENT":
			inEvent = true
			props = map[string]string{}
			continue
		case "END:VEVENT":
			if inEvent {
				flush()
			}
			inEvent = false
			continue
		}

		if !inEvent {
			continue
		}
		k, v := splitICalLine(line)
		if k == "" {
			continue
		}
		if old, ok := props[k]; ok && old != "" && v != "" && old != v {
			props[k] = old + "\n" + v
		} else {
			props[k] = v
		}
	}

	return events, nil
}



func publishEventsCount(jsc nats.JetStreamContext, subject string, events []Event) int {
	published := 0
	for _, e := range events {
		b, err := json.Marshal(e)
		if err != nil {
			log.Printf("[scheduler] json marshal error: %v", err)
			continue
		}
		if _, err := jsc.Publish(subject, b); err != nil {
			log.Printf("[scheduler] publish error: %v", err)
			continue
		}
		published++
	}
	return published
}



func fetchAndPublishFromConfigAndIcal(jsc nats.JetStreamContext) {
	configURL := env("CONFIG_URL", "http://localhost:8080/agendas")
	icalBase := env("ICAL_BASE",
		"https://edt.uca.fr/jsp/custom/modules/plannings/anonymous_cal.jsp?projectId=3&calType=ical&nbWeeks=100&displayConfigId=128",
	)
	subject := env("NATS_SUBJECT", "EVENTS.new")

	resp, err := http.Get(configURL)
	if err != nil {
		log.Printf("[scheduler] config http error: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	ids := parseAgendaIDsFromConfigPlainText(string(body))
	ids = filterLikelyUCAResources(ids)
	log.Printf("[scheduler] agenda IDs (filtered): %v", ids)

	if len(ids) == 0 {
		log.Printf("[scheduler] no usable agenda IDs -> skipping")
		return
	}

	totalParsed := 0
	totalPublished := 0

	for _, agendaID := range ids {
		icalURL, err := buildIcalURL(icalBase, []string{agendaID})
		if err != nil {
			log.Printf("[scheduler] build iCal URL error (id=%s): %v", agendaID, err)
			continue
		}

		resp2, err := http.Get(icalURL)
		if err != nil {
			log.Printf("[scheduler] iCal http error (id=%s): %v", agendaID, err)
			continue
		}
		rawIcal, _ := io.ReadAll(resp2.Body)
		resp2.Body.Close()

		if !bytes.Contains(rawIcal, []byte("BEGIN:VCALENDAR")) {
			log.Printf("[scheduler] iCal not VCALENDAR (id=%s) first 200: %q",
				agendaID, string(rawIcal[:min(200, len(rawIcal))]))
			continue
		}

		events, err := parseEventsFromICal(rawIcal, agendaID)
		for i := range events {
    		b, _ := json.MarshalIndent(events[i], "", "  ")
    		fmt.Println(string(b))
		}


		if err != nil {
			log.Printf("[scheduler] parse error (id=%s): %v", agendaID, err)
			continue
		}

		totalParsed += len(events)
		published := publishEventsCount(jsc, subject, events)
		totalPublished += published

		log.Printf("[scheduler] id=%s parsed=%d published=%d", agendaID, len(events), published)
	}

	log.Printf("[scheduler] done: parsed=%d published=%d", totalParsed, totalPublished)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	natsURL := env("NATS_URL", nats.DefaultURL)

	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Drain()

	jsc, err := nc.JetStream()
	if err != nil {
		log.Fatal(err)
	}

	_, err = jsc.AddStream(&nats.StreamConfig{
		Name:     env("NATS_STREAM", "EVENTS"),
		Subjects: []string{"EVENTS.>"},
	})
	if err != nil {
		log.Printf("[scheduler] AddStream: %v", err)
	}

	ctx := context.Background()
	sc := scheduler.NewScheduler()

	periodStr := env("SCHEDULER_PERIOD", "10s")
	period, err := time.ParseDuration(periodStr)
	if err != nil {
		period = 10 * time.Second
	}

	sc.Add(ctx, func(c context.Context) {
		fetchAndPublishFromConfigAndIcal(jsc)
	}, period)

	log.Println("[scheduler] running... Ctrl+C to stop")
	select {}
}

