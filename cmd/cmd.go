package cmd

import (
	"fmt"
	"sync"

	"github.com/gabriel-samfira/go-wmi/wmi"
	"github.com/go-ole/go-ole"
)

var mutex = sync.Mutex{}

type Command struct {
	cmd  string
	cwd  string
	proc *wmi.WMIResult
	conn *wmi.WMI
}

func NewCommand(server, username, password, domain, cmd string) (*Command, error) {
	var authority string
	if server == "" {
		server = "."
	}
	if domain != "" {
		authority = fmt.Sprintf("Kerberos:%s", domain)
	}
	w, err := wmi.NewConnection(server, `Root\CIMV2`, username, password, nil, authority)
	if err != nil {
		return nil, err
	}

	proc, err := w.Get("Win32_Process")
	if err != nil {
		return nil, err
	}

	return &Command{
		cmd:  cmd,
		proc: proc,
		conn: w,
	}, nil
}

func (c *Command) SetCWD(path string) {
	mutex.Lock()
	defer mutex.Unlock()
	c.cwd = path
}

func (c *Command) GetCWD() string {
	mutex.Lock()
	defer mutex.Unlock()
	return c.cwd
}

func (c *Command) Run() error {
	cwd := c.GetCWD()
	processId := ole.VARIANT{}
	ret, err := c.proc.Get("Create", c.cmd, cwd, nil, &processId)
	if err != nil {
		return fmt.Errorf("Error running Create: %v", err)
	}
	if ret.Value().(int32) != 0 {
		return fmt.Errorf("process exited with status: %v", ret.Value().(int32))
	}
	fmt.Printf("Process %v exited with status 0\r\n", processId.Value().(int32))
	return nil
}
