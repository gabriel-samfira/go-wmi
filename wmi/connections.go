package wmi

// NewStandardCimV2Connection returns a new WMI connection
// to the Root\StandardCimv2 namespace.
func NewStandardCimV2Connection() (w *WMI, err error) {
	return NewConnection(".", `Root\StandardCimv2`)
}
