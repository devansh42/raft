package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
)

func startSvc(port int) {

	rpc.Register(&Server{
		CurState: Follower,
	})
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal(err)
	}

	go http.Serve(l, nil)
	log.Print("Server started at ", port)
}

func main() {
	var port *int
	port = flag.Int("port", 8080, "default port")
	flag.Parse()
	for !flag.Parsed() {
	}
	startSvc(*port)
}
