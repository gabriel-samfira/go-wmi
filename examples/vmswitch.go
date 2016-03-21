package main

import (
	"fmt"
	"os"

	virt "github.com/gabriel-samfira/go-wmi/virt"
	// wmi "github.com/gabriel-samfira/go-wmi/wmi"
)

func main() {
	swname := "br100"

	vmsw, err := virt.NewVmSwitch(swname)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	name := "Intel(R) PRO/1000 MT Network Connection #2"

	fmt.Printf("Creating %s\r\n", swname)
	if err := vmsw.Create(); err != nil {
		fmt.Println(err)
		return
	}

	// newName := "newName"

	// fmt.Printf("Setting VMswitch name to: %s\r\n", newName)
	// if err := vmsw.SetSwitchName(newName); err != nil {
	// 	fmt.Println(err)
	// 	return
	// }

	fmt.Printf("Setting external port to: %s\r\n", name)
	if err := vmsw.SetExternalPort(name); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Removing ports from %s\r\n", vmsw.Name())
	if err := vmsw.RemovePort(); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("deleting %s\r\n", vmsw.Name())
	if err := vmsw.Delete(); err != nil {
		fmt.Println(err)
		return
	}
	vmsw.Release()
}
