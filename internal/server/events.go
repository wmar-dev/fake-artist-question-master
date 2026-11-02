package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"fakeartist/internal/game"
)

const (
	keepAliveEvery = 25 * time.Second
	writeTimeout   = 10 * time.Second
)

// eventStream writes Server-Sent Events to one browser.
type eventStream struct {
	w http.ResponseWriter
	c *http.ResponseController
}

func (e eventStream) write(text string) error {
	// An unsupported deadline only means a stuck client lingers longer.
	_ = e.c.SetWriteDeadline(time.Now().Add(writeTimeout))
	if _, err := fmt.Fprint(e.w, text); err != nil {
		return err
	}
	return e.c.Flush()
}

func (e eventStream) send(name string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return e.write("event: " + name + "\ndata: " + string(data) + "\n\n")
}

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	g, playerID, err := s.authenticate(r)
	if err != nil {
		writeError(w, err)
		return
	}
	sub, err := g.Subscribe(playerID)
	if err != nil {
		writeError(w, err)
		return
	}
	defer g.Unsubscribe(sub)

	header := w.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-cache")
	header.Set("X-Accel-Buffering", "no")
	stream := eventStream{w: w, c: http.NewResponseController(w)}

	keepAlive := make(chan struct{}, 1)
	schedule := func() game.Timer {
		return s.clock.AfterFunc(keepAliveEvery, func() { keepAlive <- struct{}{} })
	}
	timer := schedule()
	defer func() { timer.Stop() }()

	sendState := func() (bool, error) {
		view, err := g.View(playerID)
		if err != nil {
			return false, nil // The player or game is gone; Done carries the reason.
		}
		view.JoinURL, view.JoinURLReachable = s.joinAddress(r, g.Code())
		return true, stream.send("state", view)
	}
	sendClosed := func() {
		<-sub.Done()
		stream.send("closed", map[string]string{"reason": sub.Reason()})
	}

	if ok, err := sendState(); err != nil {
		return
	} else if !ok {
		sendClosed()
		return
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case <-sub.Done():
			sendClosed()
			return
		case <-sub.Updates():
			if ok, err := sendState(); err != nil {
				return
			} else if !ok {
				sendClosed()
				return
			}
		case <-keepAlive:
			timer = schedule()
			if err := stream.write(": keep-alive\n\n"); err != nil {
				return
			}
		}
	}
}
