package main

import (
	"gtas/cli"
	"runtime"
)

func main() {
	runtime.GOMAXPROCS(4)
	gtas := cli.NewGtas()
	gtas.Run()
}

