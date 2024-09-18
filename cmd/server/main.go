package main

import (
	"fmt"
	"godistrserv/internal/server"
	"log"
)

func main() {
	srv := server.NewHTTPServer(":8080")
	fmt.Println("start")
	log.Fatal(srv.ListenAndServe())
}
