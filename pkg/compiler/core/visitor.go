// Copyright 2016 Marapongo, Inc. All rights reserved.

package core

import (
	"github.com/marapongo/mu/pkg/ast"
	"github.com/marapongo/mu/pkg/diag"
)

// Visitor unifies all visitation patterns under a single interface.
type Visitor interface {
	Phase
	VisitMetadata(doc *diag.Document, kind string, meta *ast.Metadata)
	VisitStack(doc *diag.Document, stack *ast.Stack)
	VisitParameter(doc *diag.Document, name string, param *ast.Parameter)
	VisitDependency(doc *diag.Document, name ast.Name, dep *ast.Dependency)
	VisitServices(doc *diag.Document, svcs *ast.Services)
	VisitService(doc *diag.Document, name ast.Name, public bool, svc *ast.Service)
	VisitTarget(doc *diag.Document, name string, target *ast.Target)
}

// NewInOrderVisitor wraps another Visitor and walks the tree in a deterministic order, deferring to another set of
// Visitor objects for pre- and/or post-actions.  Either pre or post may be nil.
func NewInOrderVisitor(pre Visitor, post Visitor) Visitor {
	return &inOrderVisitor{pre, post}
}

// inOrderVisitor simply implements the Visitor pattern as specified above.
//
// Note that we need to iterate all maps in a stable order (since Go's are unordered by default).  Sadly, this
// is rather verbose due to Go's lack of generics, reflectionless Keys() functions, and so on.
type inOrderVisitor struct {
	pre  Visitor
	post Visitor
}

func (v *inOrderVisitor) Diag() diag.Sink {
	if v.pre != nil {
		return v.pre.Diag()
	}
	if v.post != nil {
		return v.post.Diag()
	}
	return nil
}

func (v *inOrderVisitor) VisitMetadata(doc *diag.Document, kind string, meta *ast.Metadata) {
	if v.pre != nil {
		v.pre.VisitMetadata(doc, kind, meta)
	}

	for _, name := range ast.StableTargets(meta.Targets) {
		target := meta.Targets[name]
		v.VisitTarget(doc, name, &target)
		// Copy the targeteter back in case it was updated.
		meta.Targets[name] = target
	}

	if v.post != nil {
		v.post.VisitMetadata(doc, kind, meta)
	}
}

func (v *inOrderVisitor) VisitStack(doc *diag.Document, stack *ast.Stack) {
	if v.pre != nil {
		v.pre.VisitStack(doc, stack)
	}

	v.VisitMetadata(doc, "Stack", &stack.Metadata)

	for _, name := range ast.StableParameters(stack.Parameters) {
		param := stack.Parameters[name]
		v.VisitParameter(doc, name, &param)
		// Copy the parameter back in case it was updated.
		stack.Parameters[name] = param
	}

	for _, name := range ast.StableDependencies(stack.Dependencies) {
		aname := ast.Name(name)
		dep := stack.Dependencies[aname]
		v.VisitDependency(doc, aname, &dep)
		// Copy the dependency back in case it was updated.
		stack.Dependencies[aname] = dep
	}

	v.VisitServices(doc, &stack.Services)

	if v.post != nil {
		v.post.VisitStack(doc, stack)
	}
}

func (v *inOrderVisitor) VisitParameter(doc *diag.Document, name string, param *ast.Parameter) {
	if v.pre != nil {
		v.pre.VisitParameter(doc, name, param)
	}
	if v.post != nil {
		v.post.VisitParameter(doc, name, param)
	}
}

func (v *inOrderVisitor) VisitDependency(doc *diag.Document, name ast.Name, dep *ast.Dependency) {
	if v.pre != nil {
		v.pre.VisitDependency(doc, name, dep)
	}
	if v.post != nil {
		v.post.VisitDependency(doc, name, dep)
	}
}

func (v *inOrderVisitor) VisitServices(doc *diag.Document, svcs *ast.Services) {
	for _, name := range ast.StableServices(svcs.Public) {
		aname := ast.Name(name)
		public := svcs.Public[aname]
		v.VisitService(doc, aname, true, &public)
		// Copy the public service back in case it was updated.
		svcs.Public[aname] = public
	}

	for _, name := range ast.StableServices(svcs.Private) {
		aname := ast.Name(name)
		private := svcs.Private[aname]
		v.VisitService(doc, aname, false, &private)
		// Copy the private service back in case it was updated.
		svcs.Private[aname] = private
	}
}

func (v *inOrderVisitor) VisitService(doc *diag.Document, name ast.Name, public bool, svc *ast.Service) {
	if v.pre != nil {
		v.pre.VisitService(doc, name, public, svc)
	}
	if v.post != nil {
		v.post.VisitService(doc, name, public, svc)
	}
}

func (v *inOrderVisitor) VisitTarget(doc *diag.Document, name string, target *ast.Target) {
	if v.pre != nil {
		v.pre.VisitTarget(doc, name, target)
	}
	if v.post != nil {
		v.post.VisitTarget(doc, name, target)
	}
}
