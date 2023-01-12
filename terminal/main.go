package main

import "github.com/solarlune/tetraterm"

func main() {

	tapp := tetraterm.NewDisplay(nil)

	err := tapp.Start()

	defer tapp.Stop()

	if err != nil {
		panic(err)
	}

}
