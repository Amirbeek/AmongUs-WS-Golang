package main

const (
	EVJoin      string = "join"       // server → clients (system)
	EVLeave     string = "leave"      // server → clients (system)
	EVChat      string = "chat"       // bi‑dir
	EVReady     string = "ready"      // client → server
	EVPhase     string = "phase"      // server → clients
	EVState     string = "state"      // server → clients (snapshot)
	EVVoteStart string = "vote_start" // server → clients
	EVVoteEnd   string = "vote_end"   // server → clients
	EVEnd       string = "end"        // server → clients
)
