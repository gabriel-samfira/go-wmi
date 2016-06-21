package main

import (
	"fmt"
	"os"

	"github.com/gabriel-samfira/go-wmi/cmd"
)

func main() {
	cmd := cmd.NewCommand("192.168.200.105","Administrator","P@ssw0rd","",`cmd.exe /c dir`)
	cmd.SetCWD(`c:\`)
	cmd.Run()
	err := cmd.Error()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
