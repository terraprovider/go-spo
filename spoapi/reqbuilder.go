package spoapi

import (
	"context"
	"fmt"
	"strings"
)

// reqBuilder assembles a ProcessQuery from ordered operations, assigning ids from a
// single counter in execution order (matching the CSOM runtime): each
// object-producing op takes a decl id then its <ObjectPath> action id; each
// SetProperty takes one id. Object paths go to the ObjectPaths section, actions to
// the Actions section, but ids interleave in add-order.
type reqBuilder struct {
	counter int
	actions strings.Builder
	paths   strings.Builder
}

// objRef is a handle to an object path in the request graph.
type objRef struct{ declID int }

func newReqBuilder() *reqBuilder { return &reqBuilder{counter: 1} }

func (b *reqBuilder) next() int { n := b.counter; b.counter++; return n }

// objectPathAction emits the <ObjectPath> action that instantiates an object path.
func (b *reqBuilder) objectPathAction(declID int) {
	fmt.Fprintf(&b.actions, `<ObjectPath Id="%d" ObjectPathId="%d" />`, b.next(), declID)
}

// constructor adds a <Constructor> object path.
func (b *reqBuilder) constructor(typeID string) *objRef {
	id := b.next()
	fmt.Fprintf(&b.paths, `<Constructor Id="%d" TypeId="%s" />`, id, typeID)
	b.objectPathAction(id)
	return &objRef{declID: id}
}

// staticMethod adds a <StaticMethod> object path (a static method on a type).
func (b *reqBuilder) staticMethod(typeID, name string, params []Param) (*objRef, error) {
	ps, err := renderParams(params)
	if err != nil {
		return nil, err
	}
	id := b.next()
	if ps == "" {
		fmt.Fprintf(&b.paths, `<StaticMethod Id="%d" Name="%s" TypeId="%s" />`, id, attrEscape(name), typeID)
	} else {
		fmt.Fprintf(&b.paths, `<StaticMethod Id="%d" Name="%s" TypeId="%s">%s</StaticMethod>`, id, attrEscape(name), typeID, ps)
	}
	b.objectPathAction(id)
	return &objRef{declID: id}, nil
}

// method adds a <Method> object path (instance method on parent); params may include
// object-path references (Param.Ref).
func (b *reqBuilder) method(parent *objRef, name string, params []refParam) (*objRef, error) {
	ps, err := renderRefParams(params)
	if err != nil {
		return nil, err
	}
	id := b.next()
	if ps == "" {
		fmt.Fprintf(&b.paths, `<Method Id="%d" ParentId="%d" Name="%s" />`, id, parent.declID, attrEscape(name))
	} else {
		fmt.Fprintf(&b.paths, `<Method Id="%d" ParentId="%d" Name="%s">%s</Method>`, id, parent.declID, attrEscape(name), ps)
	}
	b.objectPathAction(id)
	return &objRef{declID: id}, nil
}

// setProp adds a <SetProperty> action on obj.
func (b *reqBuilder) setProp(obj *objRef, sp SetProp) error {
	param, err := renderParam(sp.Type, sp.Value, sp.ElemType)
	if err != nil {
		return fmt.Errorf("spoapi: property %q: %w", sp.Name, err)
	}
	fmt.Fprintf(&b.actions, `<SetProperty Id="%d" ObjectPathId="%d" Name="%s">%s</SetProperty>`, b.next(), obj.declID, attrEscape(sp.Name), param)
	return nil
}

// query adds a <Query SelectAllProperties> action on obj and returns its id (the
// response key). childItems adds a <ChildItemQuery> for collections.
func (b *reqBuilder) query(obj *objRef, childItems bool) int {
	id := b.next()
	child := ""
	if childItems {
		child = `<ChildItemQuery SelectAllProperties="true"><Properties /></ChildItemQuery>`
	}
	fmt.Fprintf(&b.actions, `<Query Id="%d" ObjectPathId="%d"><Query SelectAllProperties="true"><Properties /></Query>%s</Query>`, id, obj.declID, child)
	return id
}

func (b *reqBuilder) build(appName string) string {
	var out strings.Builder
	fmt.Fprintf(&out, requestHeader, attrEscape(appName))
	out.WriteString("<Actions>")
	out.WriteString(b.actions.String())
	out.WriteString("</Actions><ObjectPaths>")
	out.WriteString(b.paths.String())
	out.WriteString("</ObjectPaths></Request>")
	return out.String()
}

// refParam is a method parameter that is either a value (Value) or a reference to
// another object path in the request (Ref) — rendered <Parameter ObjectPathId="…" />.
type refParam struct {
	Value *Param
	Ref   *objRef
}

func renderRefParams(params []refParam) (string, error) {
	if len(params) == 0 {
		return "", nil
	}
	var b strings.Builder
	b.WriteString("<Parameters>")
	for _, p := range params {
		if p.Ref != nil {
			fmt.Fprintf(&b, `<Parameter ObjectPathId="%d" />`, p.Ref.declID)
			continue
		}
		s, err := renderParamValue(*p.Value)
		if err != nil {
			return "", err
		}
		b.WriteString(s)
	}
	b.WriteString("</Parameters>")
	return b.String(), nil
}

// StaticUpdate describes the update pattern the -SPO* object cmdlets use: resolve an
// existing server object via a static getter (e.g. Tenant.GetSiteScript(id)), set
// properties on it, then pass it by reference to an update method (e.g.
// Tenant.UpdateSiteScript(obj)). Set-SPOSiteScript/SiteDesign/… all follow this.
type StaticUpdate struct {
	CmdletName   string    // correlation/logging
	RootTypeID   string    // the root object type (Tenant); also the static getter's type
	GetMethod    string    // static getter returning the object (e.g. "GetSiteScript")
	GetParams    []Param   // static getter params (e.g. the Guid id)
	Set          []SetProp // property assignments on the resolved object
	UpdateMethod string    // instance update method on the root (e.g. "UpdateSiteScript")
}

// InvokeStaticUpdate builds and sends a StaticUpdate. Byte-matches the runtime's wire.
func (c *Client) InvokeStaticUpdate(ctx context.Context, u StaticUpdate) (*Result, error) {
	xml, err := buildStaticUpdate(u, c.appName)
	if err != nil {
		return nil, err
	}
	return c.invokeXML(ctx, u.CmdletName, xml, 0)
}

// InvokeStaticGet resolves an object via a static getter on the root type and
// query-alls it — e.g. Tenant.GetSiteDesign(id). Returns the single object.
func (c *Client) InvokeStaticGet(ctx context.Context, cmdletName, rootTypeID, method string, params []Param) (*Result, error) {
	b := newReqBuilder()
	b.constructor(rootTypeID)
	obj, err := b.staticMethod(rootTypeID, method, params)
	if err != nil {
		return nil, err
	}
	qid := b.query(obj, false)
	return c.invokeXML(ctx, cmdletName, b.build(c.appName), qid)
}

func buildStaticUpdate(u StaticUpdate, appName string) (string, error) {
	if u.RootTypeID == "" {
		return "", fmt.Errorf("spoapi: StaticUpdate.RootTypeID is required")
	}
	b := newReqBuilder()
	root := b.constructor(u.RootTypeID)
	obj, err := b.staticMethod(u.RootTypeID, u.GetMethod, u.GetParams)
	if err != nil {
		return "", err
	}
	for _, sp := range u.Set {
		if err := b.setProp(obj, sp); err != nil {
			return "", err
		}
	}
	if _, err := b.method(root, u.UpdateMethod, []refParam{{Ref: obj}}); err != nil {
		return "", err
	}
	return b.build(appName), nil
}
