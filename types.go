package AmongUs_WS_Golang

type Envelope struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type PlayerSnap struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Alive bool   `json:"alive"`
	Ready bool   `json:"ready"`
}

type StateOut struct {
	Room    string       `json:"room"`
	Players []PlayerSnap `json:"players"`
}
