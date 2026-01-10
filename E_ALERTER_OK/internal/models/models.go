package models


type Change struct {
	Field string `json:"field"` // location,start,end,title,description
	Old   string `json:"old"`
	New   string `json:"new"`
	Kind  string `json:"kind"` // room_change,time_change, etc.
}

type AlertEvent struct {
	Type       string   `json:"type"`       // event_changed | event_new
	EventID    string   `json:"eventId"`
	AgendaIDs  []string `json:"agendaIds"`
	Title      string   `json:"title"`
	Start      string   `json:"start"`
	End        string   `json:"end"`
	Location   string   `json:"location"`
	Changes    []Change `json:"changes"`
	EmailText  string   `json:"emailText"`  
}


type AlertSubscription struct {
	ID        string `json:"id"`
	AgendaID  string `json:"agenda_id"` 
	Target    string `json:"target"`    
	Condition string `json:"condition"` // always | room_change | time_change | ...
}


//new
type OutgoingMail struct {
    Recipient string `json:"recipient"`
    Subject   string `json:"subject"`
    Content   string `json:"content"`
}









