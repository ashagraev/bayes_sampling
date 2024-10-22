package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func ServeHTTPRequests(h *Handler, port *string) {
	http.Handle("/", http.HandlerFunc(h.HandleHealth))
	http.Handle("/health", http.HandlerFunc(h.HandleHealth))
	http.Handle("/sample", http.HandlerFunc(h.HandleSample))
	http.Handle("/ctr", http.HandlerFunc(h.HandleGetCTR))
	http.Handle("/add_view", http.HandlerFunc(h.HandleAddView))
	http.Handle("/add_click", http.HandlerFunc(h.HandleAddClick))
	http.Handle("/set_views", http.HandlerFunc(h.HandleSetViews))
	http.Handle("/set_clicks", http.HandlerFunc(h.HandleSetClicks))
	http.Handle("/distribution_params", http.HandlerFunc(h.HandleReportBetaDistributionParams))
	err := http.ListenAndServe(":"+*port, nil)
	if err != nil {
		log.Fatalf("fatal error in ListenAndServe: " + err.Error())
	}
}

func main() {
	port := flag.String("port", "80", "bing the HTTP server to this port")
	flag.Parse()

	ctx := context.Background()
	h, err := NewHandler(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	go ServeHTTPRequests(h, port)

	exitSignal := make(chan os.Signal)
	signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
	<-exitSignal
}
