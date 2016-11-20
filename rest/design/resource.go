package design

import (
	"fmt"
	"path"
	"strings"

	"github.com/dimfeld/httppath"
	"goa.design/goa.v2/design"
)

type (
	// ResourceExpr describes a REST resource.
	// It defines both a media type and a set of actions that can be executed through HTTP
	// requests.
	// ResourceExpr embeds a ServiceExpr and adds HTTP specific properties.
	ResourceExpr struct {
		// ServiceExpr is the underlying service.
		*design.ServiceExpr
		// Default media type identifier.
		MediaType string
		// Schemes is the supported HTTP schemes.
		Schemes []string
		// Common URL prefix to all resource action HTTP requests
		BasePath string
		// Name of parent resource if any
		ParentName string
		// Action with canonical resource path
		CanonicalActionName string
		// Path and query string parameters that apply to all actions.
		Params *design.AttributeExpr
		// Request headers that apply to all actions.
		Headers *design.AttributeExpr
		// Actions is the list of resource actions.
		Actions []*ActionExpr
		// Responses lists HTTP responses that apply to all actions.
		Responses []*HTTPResponseExpr
		// FileServers is the list of static asset serving endpoints
		FileServers []*FileServerExpr
	}

	// ResourceIterator is the type of functions given to IterateResources.
	ResourceIterator func(r *ResourceExpr) error
)

// NewResourceExpr creates a resource definition but does not
// execute the DSL.
func NewResourceExpr(service *design.ServiceExpr) *ResourceExpr {
	mt := "text/plain"
	if dmt, ok := service.DefaultType.(*design.MediaTypeExpr); ok {
		mt = dmt.Identifier
	}
	return &ResourceExpr{
		ServiceExpr: service,
		MediaType:   mt,
	}
}

// EvalName returns the generic definition name used in error messages.
func (r *ResourceExpr) EvalName() string {
	if r.Name != "" {
		return fmt.Sprintf("resource %#v", r.Name)
	}
	return "unnamed resource"
}

// IterateHeaders calls the given iterator passing in each response sorted in alphabetical order.
// Iteration stops if an iterator returns an error and in this case IterateHeaders returns that
// error.
func (r *ResourceExpr) IterateHeaders(it HeaderIterator) error {
	return iterateHeaders(r.Headers, r.Headers.IsRequired, it)
}

// Action returns the resource action with the given name or nil if there isn't one.
func (r *ResourceExpr) Action(name string) *ActionExpr {
	for _, a := range r.Actions {
		if a.Name == name {
			return a
		}
	}
	return nil
}

// CanonicalAction returns the canonical action of the resource if any.
// The canonical action is used to compute hrefs to resources.
func (r *ResourceExpr) CanonicalAction() *ActionExpr {
	name := r.CanonicalActionName
	if name == "" {
		name = "show"
	}
	return r.Action(name)
}

// URITemplate returns a URI template to this resource.
// The result is the empty string if the resource does not have a "show" action
// and does not define a different canonical action.
func (r *ResourceExpr) URITemplate() string {
	ca := r.CanonicalAction()
	if ca == nil || len(ca.Routes) == 0 {
		return ""
	}
	return ca.Routes[0].FullPath()
}

// FullPath computes the base path to the resource actions concatenating the API and parent resource
// base paths as needed.
func (r *ResourceExpr) FullPath() string {
	if strings.HasPrefix(r.BasePath, "//") {
		return httppath.Clean(r.BasePath)
	}
	var basePath string
	if p := r.Parent(); p != nil {
		if ca := p.CanonicalAction(); ca != nil {
			if routes := ca.Routes; len(routes) > 0 {
				// Note: all these tests should be true at code generation time
				// as DSL validation makes sure that parent resources have a
				// canonical path.
				basePath = path.Join(routes[0].FullPath())
			}
		}
	} else {
		basePath = Root.BasePath
	}
	return httppath.Clean(path.Join(basePath, r.BasePath))
}

// Parent returns the parent resource if any, nil otherwise.
func (r *ResourceExpr) Parent() *ResourceExpr {
	if r.ParentName != "" {
		if parent := Root.Resource(r.ParentName); parent != nil {
			return parent
		}
	}
	return nil
}

// Response returns the resource response with given name if any.
func (r *ResourceExpr) Response(name string) *HTTPResponseExpr {
	for _, resp := range r.Responses {
		if resp.Name == name {
			return resp
		}
	}
	return nil
}

// Finalize is run post DSL execution. It merges response definitions, creates implicit action
// parameters, initializes querystring parameters, sets path parameters as non zero attributes
// and sets the fallbacks for security schemes.
func (r *ResourceExpr) Finalize() {
	for _, f := range r.FileServers {
		f.Finalize()
	}
	for _, a := range r.Actions {
		a.Finalize()
	}
}