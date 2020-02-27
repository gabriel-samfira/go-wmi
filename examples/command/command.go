package main

import (
	"fmt"

	"github.com/gabriel-samfira/go-wmi/cmd"
)

func main() {
	c, err := cmd.NewCommand("192.168.200.105", "Administrator", "P@ssw0rd", "", `cmd.exe /c dir`)
	if err != nil {
		fmt.Println(err)
		return
	}
	c.SetCWD(`c:\`)
	err = c.Run()
	if err != nil {
		fmt.Println(err)
	}
	return
}
