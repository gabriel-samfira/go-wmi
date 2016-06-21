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
	err  error
}

func (c Command) Error() error {
	return c.err
}

func NewCommand(server, username, password, domain, cmd string) *Command {
	var (
		err       error
		authority string
	)

	command := &Command{}
	command.cmd = cmd

	// var authority string
	if server == "" {
		server = "."
	}

	if domain != "" {
		authority = fmt.Sprintf("Kerberos:%s", domain)
	}

	command.conn, err = wmi.NewConnection(server, `Root\CIMV2`, username, password, authority, nil)
	if err != nil {
		command.err = err
		return command
	}

	command.proc, err = command.conn.Get("Win32_Process")
	if err != nil {
		command.err = err
		return command
	}

	return command
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

func (c *Command) Run() {
	if c.err != nil {
		return
	}

	cwd := c.GetCWD()
	processID := ole.VARIANT{}

	ret, err := c.proc.Get("Create", c.cmd, cwd, nil, &processID)
	if err != nil {
		c.err = fmt.Errorf("Error running Create: %v", err)
		return
	}

	if ret.Value().(int32) != 0 {
		c.err = fmt.Errorf("process exited with status: %v", ret.Value().(int32))
		return
	}

	fmt.Printf("Process %v exited with status 0\r\n", processID.Value().(int32))
}
