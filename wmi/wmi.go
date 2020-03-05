package wmi

import (
	"fmt"
	"strings"
	"sync"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"github.com/pkg/errors"
)

// QueryType holds the condition of the query
type QueryType string

var (
	// Equals is the qeual conditional for a query
	Equals QueryType = "="
	// Like is the pattern match conditional of a query
	Like  QueryType = " Like "
	mutex           = sync.RWMutex{}
)

// WMI represents a WMI connection object
type WMI struct {
	rawSvc     *ole.VARIANT
	unknown    *ole.IUnknown
	wmi        *ole.IDispatch
	qInterface *ole.IDispatch

	Namespace string
	Server    string

	params []interface{}
}

// NewResult wraps an ole.VARINT in a *Result
func NewResult(v *ole.VARIANT) *Result {
	return &Result{
		rawRes: v,
	}
}

// Result holds the raw WMI result of a query
type Result struct {
	rawRes *ole.VARIANT

	err error
}

// Query is an interface that defines a query type
type Query interface {
	AsString(partialQuery string) (string, error)
}

// QueryFields is a helper structure that enables us to build queries
type QueryFields struct {
	Key   string
	Value interface{}
	Type  QueryType
}

func (w *QueryFields) sanitizeValue(val interface{}) (string, error) {
	switch v := val.(type) {
	case int, string, bool, float32, float64, int64, int32:
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

// AndQuery defines an "AND" conditional query.
type AndQuery struct {
	QueryFields
}

// AsString implements the Query interface
func (w *AndQuery) AsString(partialQuery string) (string, error) {
	return w.buildQuery(partialQuery, "AND")
}

// OrQuery defines an "OR" conditional query
type OrQuery struct {
	QueryFields
}

// AsString implements the Query interface
func (w *OrQuery) AsString(partialQuery string) (string, error) {
	return w.buildQuery(partialQuery, "OR")
}

func (r *Result) setError(err error) {
	mutex.Lock()
	defer mutex.Unlock()
	r.err = err
}

func (r *Result) Error() error {
	mutex.Lock()
	defer mutex.Unlock()
	return r.err
}

// Raw returns the raw WMI result
func (r *Result) Raw() *ole.VARIANT {
	return r.rawRes
}

// ItemAtIndex returns the result of the ItemIndex WMI call on a
// raw WMI result object
func (r *Result) ItemAtIndex(i int) (*Result, error) {
	res := r.rawRes.ToIDispatch()
	if res == nil {
		return nil, fmt.Errorf("Object is not callable")
	}
	res.AddRef()
	defer res.Release()

	itemRaw, err := oleutil.CallMethod(res, "ItemIndex", i)
	if err != nil {
		return nil, errors.Wrap(err, "ItemIndex")
	}
	wmiRes := &Result{
		rawRes: itemRaw,
	}
	return wmiRes, nil
}

// Elements returns an array of WMI results
func (r *Result) Elements() ([]*Result, error) {
	var err error
	var count int
	count, err = r.Count()
	if err != nil {
		return []*Result{}, errors.Wrap(err, "getting result count")
	}
	results := make([]*Result, count)
	for i := 0; i < count; i++ {
		results[i], err = r.ItemAtIndex(i)
		if err != nil {
			return []*Result{}, errors.Wrap(err, "ItemAtIndex")
		}
	}
	return results, nil
}

// GetProperty will return a *Result holding a given property
func (r *Result) GetProperty(property string) (*Result, error) {
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
	wmiRes := &Result{
		rawRes: rawVal,
	}
	return wmiRes, nil
}

// Get will execute a method on the WMI object held in *Result, with the given params
func (r *Result) Get(method string, params ...interface{}) (*Result, error) {
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
	wmiRes := &Result{
		rawRes: rawSvc,
	}
	return wmiRes, nil
}

// Path returns the Path element of this WMI object
func (r *Result) Path() (string, error) {
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

// Set will set the parameters of a property
func (r *Result) Set(property string, params ...interface{}) error {
	res := r.rawRes.ToIDispatch()
	if res == nil {
		return fmt.Errorf("Object is not callable")
	}
	res.AddRef()
	defer res.Release()
	_, err := oleutil.PutProperty(res, property, params...)
	return err
}

// GetText returns an XML representation of an object or instance
func (r *Result) GetText(i int) (string, error) {
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

// Value returns the value of a result as an interface. It is the job
// of the caller to cast it to it's proper type
func (r *Result) Value() interface{} {
	if r == nil || r.rawRes == nil {
		return ""
	}
	return r.rawRes.Value()
}

// ToArray returna a *ole.SafeArrayConversion from the WMI result.
// This should probably not be exposed directly.
func (r *Result) ToArray() *ole.SafeArrayConversion {
	if r == nil || r.rawRes == nil {
		return nil
	}
	return r.rawRes.ToArray()
}

// Count returns the total number of results returned by the query.
func (r *Result) Count() (int, error) {
	res := r.rawRes.ToIDispatch()
	if res == nil {
		return 0, nil
	}
	countVar, err := oleutil.GetProperty(res, "Count")
	if err != nil {
		return 0, errors.Wrap(err, "getting Count property")
	}
	return int(countVar.Val), nil
}

// NewWMIObject returns a new *Result from a path
func NewWMIObject(path string) (*Result, error) {
	return nil, nil
}

// NewConnection returns a new *WMI connection, given the parameters
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
		return nil, fmt.Errorf("Error: %v", err)
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

// Close will close the WMI connection and release all resources.
func (w *WMI) Close() {
	w.wmi.Release()
	w.qInterface.Release()
	w.unknown.Release()
	ole.CoUninitialize()
}

func (w *WMI) getQueryParams(qParams []Query) (string, error) {
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

// Get returns a new *Result, given the params
func (w *WMI) Get(params ...interface{}) (*Result, error) {
	rawSvc, err := oleutil.CallMethod(w.wmi, "Get", params...)
	if err != nil {
		return nil, err
	}
	ret := &Result{
		rawRes: rawSvc,
	}
	return ret, nil
}

// ExecMethod wraps the WMI ExecMethod call and returns a *Result
func (w *WMI) ExecMethod(params ...interface{}) (*Result, error) {
	rawSvc, err := oleutil.CallMethod(w.wmi, "ExecMethod", params...)
	if err != nil {
		return nil, err
	}
	ret := &Result{
		rawRes: rawSvc,
	}
	return ret, nil
}

// Gwmi makes a WMI query and returns a *Result
func (w *WMI) Gwmi(resource string, fields []string, qParams []Query) (*Result, error) {
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
	wmiRes := &Result{
		rawRes: resultRaw,
	}
	return wmiRes, nil
}

// GetOne returns the first result from a query response.
func (w *WMI) GetOne(resource string, fields []string, qParams []Query) (*Result, error) {
	res, err := w.Gwmi(resource, fields, qParams)
	if err != nil {
		return nil, err
	}
	c, err := res.Count()
	if err != nil {
		return nil, err
	}
	if c == 0 {
		return nil, ErrNotFound
	}
	item, err := res.ItemAtIndex(0)
	if err != nil {
		return nil, err
	}
	return item, nil
}
