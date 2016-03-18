package wmi

import (
	"fmt"
	"strings"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

type WMIQueryType string
type SetParams interface{}

var (
	Equals WMIQueryType = "="
	Like   WMIQueryType = " Like "
)

type WMI struct {
	rawSvc     *ole.VARIANT
	unknown    *ole.IUnknown
	wmi        *ole.IDispatch
	qInterface *ole.IDispatch

	Namespace string
	Server    string

	params []interface{}
}

type WMIResult struct {
	rawRes *ole.VARIANT
}

type WMIQuery interface {
	AsString(partialQuery string) (string, error)
}

type QueryFields struct {
	Key   string
	Value interface{}
	Type  WMIQueryType
}

func (w *QueryFields) sanitizeValue(val interface{}) (string, error) {
	switch v := val.(type) {
	case int, string, bool, float32, float64:
		return fmt.Sprintf("%v", v), nil
	default:
		return "", fmt.Errorf("Invalid field value")
	}
}

func (w *QueryFields) validateFields() error {
	if w.Key == "" || w.Type == "" {
		return fmt.Errorf("Invalid parameters (key: %v, Type: %v, Value: %v", w.Key, w.Type, w.Value)
	}
	return nil
}

func (w *QueryFields) buildQuery(partialQuery, cond string) (string, error) {
	if err := w.validateFields(); err != nil {
		return "", err
	}
	v, err := w.sanitizeValue(w.Value)
	if err != nil {
		return "", err
	}
	if partialQuery == "" {
		partialQuery += "WHERE"
	}
	if strings.HasSuffix(partialQuery, "WHERE") {
		partialQuery += fmt.Sprintf(" %s%s'%s'", w.Key, w.Type, v)
	} else {
		partialQuery += fmt.Sprintf(" %s %s%s'%s'", cond, w.Key, w.Type, v)
	}
	return partialQuery, nil
}

type WMIAndQuery struct {
	QueryFields
}

func (w *WMIAndQuery) AsString(partialQuery string) (string, error) {
	return w.buildQuery(partialQuery, "AND")
}

type WMIOrQuery struct {
	QueryFields
}

func (w *WMIOrQuery) AsString(partialQuery string) (string, error) {
	return w.buildQuery(partialQuery, "OR")
}

func (r *WMIResult) Raw() *ole.VARIANT {
	return r.rawRes
}

func (r *WMIResult) ItemAtIndex(i int) (*WMIResult, error) {
	res := r.rawRes.ToIDispatch()
	if res == nil {
		return nil, fmt.Errorf("Object is not callable")
	}
	res.AddRef()
	defer res.Release()

	itemRaw, err := oleutil.CallMethod(res, "ItemIndex", i)
	if err != nil {
		return nil, err
	}
	wmiRes := &WMIResult{
		rawRes: itemRaw,
	}
	return wmiRes, nil
}

func (r *WMIResult) GetProperty(property string) (*WMIResult, error) {
	res := r.rawRes.ToIDispatch()
	if res == nil {
		return nil, fmt.Errorf("Object is not callable")
	}
	res.AddRef()
	defer res.Release()

	rawVal, err := oleutil.GetProperty(res, property)
	if err != nil {
		return nil, err
	}
	wmiRes := &WMIResult{
		rawRes: rawVal,
	}
	return wmiRes, nil
}

func (r *WMIResult) Get(method string, params ...interface{}) (*WMIResult, error) {
	res := r.rawRes.ToIDispatch()
	if res == nil {
		return nil, fmt.Errorf("Object is not callable")
	}
	res.AddRef()
	defer res.Release()

	rawSvc, err := oleutil.CallMethod(res, method, params...)
	if err != nil {
		return nil, err
	}
	wmiRes := &WMIResult{
		rawRes: rawSvc,
	}
	return wmiRes, nil
}

func (r *WMIResult) Path() (string, error) {
	p, err := r.GetProperty("path_")
	if err != nil {
		return "", err
	}
	path, err := p.GetProperty("Path")
	if err != nil {
		return "", err
	}
	val := path.Value()
	if val == nil {
		return "", fmt.Errorf("Failed to get Path_")
	}
	return val.(string), nil
}

func (r *WMIResult) Set(property string, params ...interface{}) error {
	res := r.rawRes.ToIDispatch()
	if res == nil {
		return fmt.Errorf("Object is not callable")
	}
	res.AddRef()
	defer res.Release()
	_, err := oleutil.PutProperty(res, property, params...)
	return err
}

func (r *WMIResult) GetText(i int) (string, error) {
	res := r.rawRes.ToIDispatch()
	if res == nil {
		return "", fmt.Errorf("Object is not callable")
	}
	res.AddRef()
	defer res.Release()
	t, err := oleutil.CallMethod(res, "GetText_", i)
	if err != nil {
		return "", err
	}
	return t.ToString(), nil
}

func (r *WMIResult) Value() interface{} {
	if r == nil || r.rawRes == nil {
		return ""
	}
	return r.rawRes.Value()
}

func (r *WMIResult) ToArray() *ole.SafeArrayConversion {
	if r == nil || r.rawRes == nil {
		return nil
	}
	return r.rawRes.ToArray()
}

func (r *WMIResult) Count() (int, error) {
	res := r.rawRes.ToIDispatch()
	if res == nil {
		return 0, nil
	}
	countVar, err := oleutil.GetProperty(res, "Count")
	if err != nil {
		return 0, err
	}
	return int(countVar.Val), nil
}

func NewWMIObject(path string) (*WMIResult, error) {
	return nil, nil
}

func NewConnection(params ...interface{}) (*WMI, error) {
	err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED)
	if err != nil {
		oleerr := err.(*ole.OleError)
		// CoInitialize already called
		// https://msdn.microsoft.com/en-us/library/windows/desktop/ms695279%28v=vs.85%29.aspx
		if oleerr.Code() != ole.S_OK && oleerr.Code() != 0x00000001 {
			return nil, err
		}
	}
	unknown, err := oleutil.CreateObject("WbemScripting.SWbemLocator")
	if err != nil {
		return nil, err
	}
	qInterface, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return nil, err
	}
	rawSvc, err := oleutil.CallMethod(qInterface, "ConnectServer", params...)
	if err != nil {
		return nil, err
	}
	wmi := rawSvc.ToIDispatch()
	w := &WMI{
		rawSvc:     rawSvc,
		unknown:    unknown,
		qInterface: qInterface,
		wmi:        wmi,
		params:     params,
	}
	return w, nil
}

func (w *WMI) Close() {
	w.wmi.Release()
	w.qInterface.Release()
	w.unknown.Release()
	ole.CoUninitialize()
}

func (w *WMI) getQueryParams(qParams []WMIQuery) (string, error) {
	if len(qParams) == 0 {
		return "", nil
	}
	var ret string
	var err error
	for _, q := range qParams {
		ret, err = q.AsString(ret)
		if err != nil {
			return "", err
		}
	}
	return ret, nil
}

func (w *WMI) Get(params ...interface{}) (*WMIResult, error) {
	rawSvc, err := oleutil.CallMethod(w.wmi, "Get", params...)
	if err != nil {
		return nil, err
	}
	ret := &WMIResult{
		rawRes: rawSvc,
	}
	return ret, nil
}

func (w *WMI) ExecMethod(params ...interface{}) (*WMIResult, error) {
	rawSvc, err := oleutil.CallMethod(w.wmi, "ExecMethod", params...)
	if err != nil {
		return nil, err
	}
	ret := &WMIResult{
		rawRes: rawSvc,
	}
	return ret, nil
}

func (w *WMI) Gwmi(resource string, fields []string, qParams []WMIQuery) (*WMIResult, error) {
	n := "*"
	if len(fields) > 0 {
		n = strings.Join(fields, ",")
	}
	qStr, err := w.getQueryParams(qParams)
	if err != nil {
		return nil, err
	}
	// result is a SWBemObjectSet
	q := fmt.Sprintf("SELECT %s FROM %s %s", n, resource, qStr)
	// fmt.Println(q)
	resultRaw, err := oleutil.CallMethod(w.wmi, "ExecQuery", q)
	if err != nil {
		return nil, err
	}
	wmiRes := &WMIResult{
		rawRes: resultRaw,
	}
	return wmiRes, nil
}

func (w *WMI) GetOne(resource string, fields []string, qParams []WMIQuery) (*WMIResult, error) {
	res, err := w.Gwmi(resource, fields, qParams)
	if err != nil {
		return nil, err
	}
	c, err := res.Count()
	if err != nil {
		return nil, err
	}
	if c == 0 {
		return nil, NotFoundError
	}
	item, err := res.ItemAtIndex(0)
	if err != nil {
		return nil, err
	}
	return item, nil
}
