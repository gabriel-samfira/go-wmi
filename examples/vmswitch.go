package main

import (
	"fmt"
	"os"

	virt "github.com/gabriel-samfira/go-wmi/virt"
	// wmi "github.com/gabriel-samfira/go-wmi/wmi"
)

func main() {
	swname := "br100"
	name := "Intel(R) PRO/1000 MT Network Connection #2"

	vmsw := virt.NewVmSwitch(swname)
	fmt.Printf("Creating %s\r\n", swname)
	vmsw.Create()

	// newName := "newName"

	// fmt.Printf("Setting VMswitch name to: %s\r\n", newName)
	// if err := vmsw.SetSwitchName(newName); err != nil {
	// 	fmt.Println(err)
	// 	return
	// }

	fmt.Printf("Setting external port to: %s\r\n", name)
	vmsw.SetExternalPort(name)

	fmt.Printf("Removing ports from %s\r\n", vmsw.Name())
	vmsw.RemovePort()

	fmt.Printf("deleting %s\r\n", vmsw.Name())
	vmsw.Delete()

	vmsw.Release()

	if err := vmsw.Error(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
