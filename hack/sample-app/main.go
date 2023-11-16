package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		slog.LogAttrs(r.Context(), slog.LevelInfo, "received request",
			slog.String("path", r.URL.Path))
		fmt.Fprintf(w, "sample-app%s", r.URL.Path)
	})
	http.ListenAndServe(":80", nil)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	// Block until a signal is received.
	<-c
}
