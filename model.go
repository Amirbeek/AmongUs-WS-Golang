package main

type Phase string

const (
	PhaseWaiting Phase = "waiting"
	PhaseInGame  Phase = "inGame"
	PhaseEnded   Phase = "ended"
)
