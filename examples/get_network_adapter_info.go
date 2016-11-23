package main

import (
	"encoding/json"
	"fmt"

	virt "github.com/gabriel-samfira/go-wmi/virt"
)

func main() {
	adapters, err := virt.GetNetworkAdapters("")
	if err != nil {
		fmt.Println(err)
		return
	}
	m, err := json.MarshalIndent(&adapters, "", "  ")
	fmt.Println(string(m), err)
}
