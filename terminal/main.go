package main

import (
	"flag"

	"github.com/solarlune/tetraterm"
)

func main() {

	hostName := flag.String("host", "", "Defines the host for TetraTerm to listen to. A blank string means localhost (this machine).")
	portNumber := flag.String("port", "7979", "Defines the port for TetraTerm to listen on. This should be the same as the server in your game.")

	flag.Parse()

	settings := tetraterm.NewDefaultConnectionSettings()

	settings.Host = *hostName
	settings.Port = *portNumber

	tapp := tetraterm.NewDisplay(settings)

	err := tapp.Start()

	defer tapp.Stop()

	if err != nil {
		panic(err)
	}

}
