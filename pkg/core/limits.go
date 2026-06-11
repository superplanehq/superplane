package core

// MaxEmitCount is the maximum number of events a single execution may emit at once.
// Components that fan out (For Each, Read Memory "One By One", and similar) must stay
// within this limit so we do not create unbounded downstream runs, DB rows, or queue load.
const MaxEmitCount = 500
