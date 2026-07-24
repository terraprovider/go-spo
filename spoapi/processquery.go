package spoapi

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// TenantTypeID is the CSOM server type identity of
// Microsoft.Online.SharePoint.TenantAdministration.Tenant, used to construct the
// tenant admin object in a ProcessQuery <ObjectPaths>. Read from the shipped CSOM
// runtime (spo-powershell-api-re/tools/emit-processquery-samples.ps1).
const TenantTypeID = "{268004ae-ef6b-4e9b-8425-127220d84719}"

// ParamType is the CSOM ProcessQuery <Parameter Type="…"> token. It mirrors the
// csomType recorded per property in the derived catalog (spec/csom-model.json), so
// a generated binding maps a property directly to its wire type.
type ParamType string

const (
	TypeBoolean  ParamType = "Boolean"
	TypeString   ParamType = "String"
	TypeInt16    ParamType = "Int16"
	TypeInt32    ParamType = "Int32"
	TypeInt64    ParamType = "Int64"
	TypeDouble   ParamType = "Double"
	TypeEnum     ParamType = "Enum"
	TypeGuid     ParamType = "Guid"
	TypeDateTime ParamType = "DateTime"
	TypeArray    ParamType = "Array"
	TypeNull     ParamType = "Null"
)

// SetProp is one property assignment applied via a <SetProperty> action. Value's Go
// type must suit Type: bool for Boolean; string for String/Guid; any integer for
// Int*/Enum; float64 for Double; a slice for Array (with ElemType set).
type SetProp struct {
	Name     string
	Type     ParamType
	Value    any
	ElemType ParamType // Array only: the <Object Type="…"> of each element
}

// Param is one CSOM <Parameter> — a scalar/array/null, or a constructed object
// (ObjectTypeID set) whose Props are serialised as <Property> children.
type Param struct {
	Type     ParamType // scalar/array/null wire type; ignored when ObjectTypeID set
	Value    any
	ElemType ParamType // Array element type

	ObjectTypeID string    // constructed-object param: <Parameter TypeId="…">
	Props        []SetProp // its properties (rendered as <Property Name Type>…)
}

// PathStep is one object-returning method in the object-path chain — e.g.
// Tenant.GetSitePropertiesByUrl(url, true) → the SiteProperties object that later
// actions target. It becomes a <Method> object path plus its <ObjectPath> action.
type PathStep struct {
	Method string
	Params []Param
}

// MethodCall is a terminal void/scalar-returning method invocation (a <Method>
// action) — e.g. Tenant.Update() or Tenant.AddTenantTheme(name, json).
type MethodCall struct {
	Name   string
	Params []Param
}

// Op describes one CSOM operation. The object-path chain starts at a Constructor
// (TypeID) and is extended by Chain (object-returning methods); the terminal
// actions (Set/Invoke/Update/Query) apply to the last object in the chain.
type Op struct {
	CmdletName string     // e.g. "Set-SPOTenant" — for correlation/logging only
	TypeID     string     // root constructor TypeId (e.g. TenantTypeID)
	Chain      []PathStep // object-returning method steps after the constructor

	Set             []SetProp    // <SetProperty> actions on the target object
	Invoke          []MethodCall // terminal <Method> actions (void/scalar)
	Update          bool         // shorthand: append a void <Method Name="Update">
	Query           bool         // append <Query SelectAllProperties="true">
	QueryChildItems bool         // collection read: add <ChildItemQuery>
}

const requestHeader = `<Request AddExpandoFieldTypeSuffix="true" SchemaVersion="15.0.0.0" LibraryVersion="16.0.0.0" ApplicationName="%s" xmlns="http://schemas.microsoft.com/sharepoint/clientquery/2009">`

// buildRequest renders op into ProcessQuery XML (verified byte-for-byte against the
// committed golden fixtures for the shapes the runtime emits). It returns the XML
// and the terminal <Query> action's Id — the response key for the read result (0
// when op has no Query).
//
// ID allocation matches the runtime: a single counter walks the object-path chain
// (each step: object-path decl id, then its <ObjectPath> action id), then the
// terminal actions. So Constructor=1/action=2, first Method=3/action=4, …, then
// SetProperty/Method/Query.
func buildRequest(op Op, appName string) (xml string, queryID int, err error) {
	if op.TypeID == "" {
		return "", 0, fmt.Errorf("spoapi: Op.TypeID is required")
	}
	// Allocate ids for the chain (constructor + method steps).
	type node struct {
		declID, actionID, parentDecl int
	}
	chain := make([]node, 1+len(op.Chain))
	counter := 1
	for i := range chain {
		chain[i].declID = counter
		counter++
		chain[i].actionID = counter
		counter++
		if i > 0 {
			chain[i].parentDecl = chain[i-1].declID
		}
	}
	targetDecl := chain[len(chain)-1].declID

	var actions, paths strings.Builder
	// <ObjectPath> action per chain step, in order.
	for _, n := range chain {
		fmt.Fprintf(&actions, `<ObjectPath Id="%d" ObjectPathId="%d" />`, n.actionID, n.declID)
	}
	// Terminal actions on the target object: SetProperty, then Invoke methods,
	// then Update, then Query.
	for _, sp := range op.Set {
		param, e := renderParam(sp.Type, sp.Value, sp.ElemType)
		if e != nil {
			return "", 0, fmt.Errorf("spoapi: property %q: %w", sp.Name, e)
		}
		fmt.Fprintf(&actions, `<SetProperty Id="%d" ObjectPathId="%d" Name="%s">%s</SetProperty>`, counter, targetDecl, attrEscape(sp.Name), param)
		counter++
	}
	for _, mc := range op.Invoke {
		ps, e := renderParams(mc.Params)
		if e != nil {
			return "", 0, e
		}
		if ps == "" {
			fmt.Fprintf(&actions, `<Method Name="%s" Id="%d" ObjectPathId="%d" />`, attrEscape(mc.Name), counter, targetDecl)
		} else {
			fmt.Fprintf(&actions, `<Method Name="%s" Id="%d" ObjectPathId="%d">%s</Method>`, attrEscape(mc.Name), counter, targetDecl, ps)
		}
		counter++
	}
	if op.Update {
		fmt.Fprintf(&actions, `<Method Name="Update" Id="%d" ObjectPathId="%d" />`, counter, targetDecl)
		counter++
	}
	if op.Query {
		queryID = counter
		child := ""
		if op.QueryChildItems {
			child = `<ChildItemQuery SelectAllProperties="true"><Properties /></ChildItemQuery>`
		}
		fmt.Fprintf(&actions, `<Query Id="%d" ObjectPathId="%d"><Query SelectAllProperties="true"><Properties /></Query>%s</Query>`, counter, targetDecl, child)
		counter++
	}

	// <ObjectPaths>: constructor + method decls.
	fmt.Fprintf(&paths, `<Constructor Id="%d" TypeId="%s" />`, chain[0].declID, op.TypeID)
	for i, step := range op.Chain {
		n := chain[i+1]
		ps, e := renderParams(step.Params)
		if e != nil {
			return "", 0, e
		}
		if ps == "" {
			fmt.Fprintf(&paths, `<Method Id="%d" ParentId="%d" Name="%s" />`, n.declID, n.parentDecl, attrEscape(step.Method))
		} else {
			fmt.Fprintf(&paths, `<Method Id="%d" ParentId="%d" Name="%s">%s</Method>`, n.declID, n.parentDecl, attrEscape(step.Method), ps)
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, requestHeader, attrEscape(appName))
	b.WriteString("<Actions>")
	b.WriteString(actions.String())
	b.WriteString("</Actions><ObjectPaths>")
	b.WriteString(paths.String())
	b.WriteString("</ObjectPaths></Request>")
	return b.String(), queryID, nil
}

// renderParams renders a method's/constructor's <Parameters> block, or "" if empty.
func renderParams(params []Param) (string, error) {
	if len(params) == 0 {
		return "", nil
	}
	var b strings.Builder
	b.WriteString("<Parameters>")
	for _, p := range params {
		s, err := renderParamValue(p)
		if err != nil {
			return "", err
		}
		b.WriteString(s)
	}
	b.WriteString("</Parameters>")
	return b.String(), nil
}

// renderParamValue renders a single <Parameter> (scalar/array/null or a
// constructed object with <Property> children).
func renderParamValue(p Param) (string, error) {
	if p.ObjectTypeID != "" {
		var b strings.Builder
		fmt.Fprintf(&b, `<Parameter TypeId="%s">`, p.ObjectTypeID)
		for _, pr := range p.Props {
			if pr.Type == TypeNull || pr.Value == nil {
				fmt.Fprintf(&b, `<Property Name="%s" Type="Null" />`, attrEscape(pr.Name))
				continue
			}
			if pr.Type == TypeArray {
				arr, err := renderArrayInner(pr.ElemType, pr.Value)
				if err != nil {
					return "", err
				}
				if arr == "" {
					fmt.Fprintf(&b, `<Property Name="%s" Type="Array" />`, attrEscape(pr.Name))
				} else {
					fmt.Fprintf(&b, `<Property Name="%s" Type="Array">%s</Property>`, attrEscape(pr.Name), arr)
				}
				continue
			}
			v, err := renderScalar(pr.Type, pr.Value)
			if err != nil {
				return "", fmt.Errorf("spoapi: property %q: %w", pr.Name, err)
			}
			fmt.Fprintf(&b, `<Property Name="%s" Type="%s">%s</Property>`, attrEscape(pr.Name), pr.Type, v)
		}
		b.WriteString(`</Parameter>`)
		return b.String(), nil
	}
	return renderParam(p.Type, p.Value, p.ElemType)
}

// renderParam serialises a scalar/array/null value as a <Parameter>.
func renderParam(t ParamType, value any, elem ParamType) (string, error) {
	if t == TypeNull || value == nil {
		return `<Parameter Type="Null" />`, nil
	}
	if t == TypeArray {
		inner, err := renderArrayInner(elem, value)
		if err != nil {
			return "", err
		}
		if inner == "" {
			return `<Parameter Type="Array" />`, nil
		}
		return fmt.Sprintf(`<Parameter Type="Array">%s</Parameter>`, inner), nil
	}
	v, err := renderScalar(t, value)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`<Parameter Type="%s">%s</Parameter>`, t, v), nil
}

// renderArrayInner renders the <Object …>…</Object> elements of an array value
// (without the enclosing <Parameter>/<Property> tag); "" for an empty array.
func renderArrayInner(elem ParamType, value any) (string, error) {
	elems := toSlice(value)
	if len(elems) == 0 {
		return "", nil
	}
	var b strings.Builder
	for _, e := range elems {
		v, err := renderScalar(elem, e)
		if err != nil {
			return "", fmt.Errorf("array element: %w", err)
		}
		fmt.Fprintf(&b, `<Object Type="%s">%s</Object>`, elem, v)
	}
	return b.String(), nil
}

// renderScalar formats a Go value as CSOM element text for the given type.
func renderScalar(t ParamType, v any) (string, error) {
	switch t {
	case TypeBoolean:
		if b, ok := v.(bool); ok {
			return strconv.FormatBool(b), nil
		}
		return "", fmt.Errorf("Boolean expects bool, got %T", v)
	case TypeString:
		return textEscape(fmt.Sprintf("%v", v)), nil
	case TypeGuid:
		return braceGuid(fmt.Sprintf("%v", v)), nil
	case TypeInt16, TypeInt32, TypeInt64, TypeEnum:
		return formatInt(v)
	case TypeDouble:
		switch n := v.(type) {
		case float64:
			return strconv.FormatFloat(n, 'G', -1, 64), nil
		case float32:
			return strconv.FormatFloat(float64(n), 'G', -1, 32), nil
		default:
			return formatInt(v)
		}
	case TypeDateTime:
		return textEscape(fmt.Sprintf("%v", v)), nil
	default:
		// Unknown/complex CSOM type — render its text form; callers should avoid.
		return textEscape(fmt.Sprintf("%v", v)), nil
	}
}

func formatInt(v any) (string, error) {
	switch n := v.(type) {
	case int:
		return strconv.FormatInt(int64(n), 10), nil
	case int32:
		return strconv.FormatInt(int64(n), 10), nil
	case int64:
		return strconv.FormatInt(n, 10), nil
	case uint, uint32, uint64:
		return fmt.Sprintf("%d", n), nil
	case float64: // JSON numbers arrive as float64
		return strconv.FormatInt(int64(n), 10), nil
	case json.Number:
		return n.String(), nil
	default:
		return "", fmt.Errorf("integer expects a number, got %T", v)
	}
}

func toSlice(v any) []any {
	switch s := v.(type) {
	case []any:
		return s
	case []string:
		out := make([]any, len(s))
		for i, x := range s {
			out[i] = x
		}
		return out
	case []int:
		out := make([]any, len(s))
		for i, x := range s {
			out[i] = x
		}
		return out
	default:
		return nil
	}
}

// braceGuid ensures a GUID is wrapped in braces, as CSOM serialises it.
func braceGuid(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
		return s
	}
	return "{" + strings.Trim(s, "{}") + "}"
}

// textEscape escapes element text the way .NET XmlWriter does (& < > and CR).
func textEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\r", "&#xD;")
	return r.Replace(s)
}

// attrEscape escapes an attribute value's content (the surrounding quotes are in
// the templates), the way .NET XmlWriter escapes attribute text.
func attrEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "\r", "&#xD;", "\n", "&#xA;", "\t", "&#x9;").Replace(s)
}
