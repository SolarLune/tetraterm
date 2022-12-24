package main

import "github.com/solarlune/tetraterm"

func main() {

	tapp := tetraterm.NewDisplay(nil)

	if err := tapp.Start(); err != nil {
		panic(err)
	}

}
