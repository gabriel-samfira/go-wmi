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

	params []interface{}
}

type WMIResult struct {
	res    *ole.IDispatch
	rawRes *ole.VARIANT
}

type WMIQuery interface{}

type WMIBaseQuery struct {
	Key   string
	Value interface{}
	Type  WMIQueryType
}

type WMIAndQuery WMIBaseQuery
type WMIOrQuery WMIBaseQuery

func (r *WMIResult) Raw() *ole.IDispatch {
	return r.res
}

func (r *WMIResult) ItemAtIndex(i int) (*WMIResult, error) {
	if r.res == nil {
		return nil, fmt.Errorf("Object not found")
	}
	itemRaw, err := oleutil.CallMethod(r.res, "ItemIndex", i)
	if err != nil {
		return nil, err
	}
	item := itemRaw.ToIDispatch()
	wmiRes := &WMIResult{
		res:    item,
		rawRes: itemRaw,
	}
	return wmiRes, nil
}

func (r *WMIResult) GetProperty(property string) (*WMIResult, error) {
	if r.res == nil {
		return nil, fmt.Errorf("Object not found")
	}
	rawVal, err := oleutil.GetProperty(r.res, property)
	if err != nil {
		return nil, err
	}
	val := rawVal.ToIDispatch()
	wmiRes := &WMIResult{
		res:    val,
		rawRes: rawVal,
	}
	return wmiRes, nil
}

func (r *WMIResult) Get(method string, params ...interface{}) (*WMIResult, error) {
	rawSvc, err := oleutil.CallMethod(r.res, method, params...)
	if err != nil {
		return nil, err
	}
	svc := rawSvc.ToIDispatch()
	wmiRes := &WMIResult{
		res:    svc,
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
	_, err := oleutil.PutProperty(r.res, property, params...)
	return err
}

func (r *WMIResult) GetText(i int) (string, error) {
	t, err := oleutil.CallMethod(r.res, "GetText_", i)
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

func (r *WMIResult) Release() {
	r.res.Release()
}

func (r *WMIResult) Count() (int, error) {
	countVar, err := oleutil.GetProperty(r.res, "Count")
	if err != nil {
		return 0, err
	}
	return int(countVar.Val), nil
}

func NewConnection(params ...interface{}) (*WMI, error) {
	ole.CoInitialize(0)
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

func (w *WMI) sanitizeValue(val interface{}) (string, error) {
	switch v := val.(type) {
	case int, string, bool, float32, float64:
		return fmt.Sprintf("%v", v), nil
	default:
		return "", fmt.Errorf("Invalid field value")
	}
}

func (w *WMI) getQueryParams(qParams []WMIQuery) (string, error) {
	if len(qParams) == 0 {
		return "", nil
	}
	ret := "WHERE"
	for _, q := range qParams {
		var mod string
		var pair string
		switch v := q.(type) {
		case WMIAndQuery:
			val, err := w.sanitizeValue(v.Value)
			if err != nil {
				return "", err
			}
			mod = "and"
			pair = fmt.Sprintf("%s%s'%s'", v.Key, v.Type, val)
		case WMIOrQuery:
			val, err := w.sanitizeValue(v.Value)
			if err != nil {
				return "", err
			}
			mod = "or"
			pair = fmt.Sprintf("%s%s'%s'", v.Key, v.Type, val)
		}
		if strings.HasSuffix(ret, "WHERE") {
			ret += fmt.Sprintf(" %s", pair)
		} else {
			ret += fmt.Sprintf(" %s %s", mod, pair)
		}
	}
	return ret, nil
}

func (w *WMI) Get(params ...interface{}) (*WMIResult, error) {
	rawSvc, err := oleutil.CallMethod(w.wmi, "Get", params...)
	if err != nil {
		return nil, err
	}
	svc := rawSvc.ToIDispatch()
	ret := &WMIResult{
		res:    svc,
		rawRes: rawSvc,
	}
	return ret, nil
}

func (w *WMI) ExecMethod(params ...interface{}) (*WMIResult, error) {
	rawSvc, err := oleutil.CallMethod(w.wmi, "ExecMethod", params...)
	if err != nil {
		return nil, err
	}
	svc := rawSvc.ToIDispatch()
	ret := &WMIResult{
		res:    svc,
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
	resultRaw, err := oleutil.CallMethod(w.wmi, "ExecQuery", q)
	if err != nil {
		return nil, err
	}
	res := resultRaw.ToIDispatch()
	wmiRes := &WMIResult{
		res:    res,
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
		return nil, fmt.Errorf("Querie returned empty set")
	}
	item, err := res.ItemAtIndex(0)
	if err != nil {
		return nil, err
	}
	return item, nil
}
