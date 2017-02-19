package main

import (
	"log"
	"net/http"
	"strconv"

	"code.myceliUs.com/CargoWebServer/Cargo/Server"
	"golang.org/x/net/websocket"
)

func main() {

	// Handle application path...
	root := Server.GetServer().GetConfigurationManager().GetApplicationDirectoryPath()
	port := Server.GetServer().GetConfigurationManager().GetServerPort()

	log.Println("Start serve files from ", root)

	// Start the web socket handler
	http.Handle("/ws", websocket.Handler(Server.HttpHandler))

	// The http handler
	http.Handle("/", http.FileServer(http.Dir(root)))

	// The file upload handler.
	http.HandleFunc("/uploads", Server.FileUploadHandler)

	// stop the server...
	defer Server.GetServer().Stop()

	// Start the server...
	Server.GetServer().Start()
	log.Println("Port:", port)
	err := http.ListenAndServe(":"+strconv.Itoa(port), nil)

	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}

}
