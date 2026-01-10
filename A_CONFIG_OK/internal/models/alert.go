package models


type Alert struct {
	ID        string `json:"id"`
	AgendaID  string `json:"agenda_id"`
	Target    string `json:"target"`    //mail
	Condition string `json:"condition"` //"always", "room_change"
}
