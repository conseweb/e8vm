package g8

import (
	"fmt"

	"e8vm.io/e8vm/build8"
	"e8vm.io/e8vm/dagvis"
	"e8vm.io/e8vm/g8/ast"
	"e8vm.io/e8vm/g8/ir"
	"e8vm.io/e8vm/g8/sempass"
	"e8vm.io/e8vm/g8/types"
	"e8vm.io/e8vm/lex8"
	"e8vm.io/e8vm/sym8"
)

type pkg struct {
	files map[string]*ast.File

	tops      *sym8.Table
	testNames []string
	deps      *dagvis.Graph
}

func newPkg(asts map[string]*ast.File) *pkg {
	ret := new(pkg)
	ret.files = asts

	return ret
}

func (p *pkg) build(b *builder, pinfo *build8.PkgInfo) []*lex8.Error {
	tops, deps, tests, errs := buildPkg(b, p.files, pinfo)
	p.tops = tops
	p.testNames = tests
	p.deps = deps
	return errs
}

func buildTests(b *builder, tops *sym8.Table) (
	testList ir.Ref, testNames []string,
) {
	tests := listTests(tops)
	n := len(tests)

	if n > 100000 {
		b.Errorf(nil, "too many tests in the package")
		return
	}

	if n == 0 {
		return
	}

	perm := b.rand.Perm(n)

	var irs []*ir.Func
	var names []string
	for _, index := range perm {
		t := tests[index]
		irs = append(irs, t.ref.IR().(*ir.Func))
		names = append(names, t.name)
	}
	return b.p.NewTestList(":tests", irs), names
}

func buildPkg(
	b *builder, files map[string]*ast.File, pinfo *build8.PkgInfo,
) (
	syms *sym8.Table, deps *dagvis.Graph,
	testNames []string, errs []*lex8.Error,
) {
	imports := make(map[string]*build8.Package)
	for as, imp := range pinfo.Import {
		imports[as] = imp.Package
	}

	sp := &sempass.Pkg{
		Path:    b.path,
		Files:   files,
		Imports: imports,
	}

	tops := sym8.NewTable()
	b.scope.PushTable(tops)
	defer b.scope.Pop()

	res, depGraph, errs := sp.Build(b.scope)
	if errs != nil {
		return nil, nil, nil, errs
	}

	for _, c := range res.Consts {
		name := c.Name()
		t := c.ObjType.(types.T)
		c.Obj = &objConst{name: name, ref: newRef(t, nil)}
	}

	for _, v := range res.Vars {
		for _, sym := range v.Left {
			t := sym.ObjType.(types.T)
			name := sym.Name()
			ref := newAddressableRef(t, b.newGlobalVar(t, name))
			sym.Obj = &objVar{name: name, ref: ref}
		}
	}

	for _, f := range res.FuncAliases {
		sym := f.Sym
		name := sym.Name()
		t := sym.ObjType.(*types.Func)
		sig := makeFuncSig(t)
		fsym := ir.NewFuncSym(f.Of.Pkg(), f.Of.Name(), sig)
		f.Sym.Obj = &objFunc{
			name:    name,
			ref:     newRef(t, fsym),
			isAlias: true,
		}
	}

	for _, f := range res.Funcs {
		name := f.Sym.Name()
		t := f.Sym.ObjType.(*types.Func)
		sig := makeFuncSig(t)
		irFunc := b.p.NewFunc(b.anonyName(name), f.Sym.Pos, sig)
		f.Sym.Obj = &objFunc{name: name, ref: newRef(t, irFunc)}
	}

	for _, f := range res.Methods {
		name := f.Sym.Name()
		t := f.Sym.ObjType.(*types.Func)
		s := t.Args[0].T.(*types.Pointer).T.(*types.Struct)

		fullName := fmt.Sprintf("%s:%s", s, name)
		sig := makeFuncSig(t)
		irFunc := b.p.NewFunc(fullName, f.Sym.Pos, sig)
		f.Sym.Obj = &objFunc{
			name:     name,
			ref:      newRef(t, irFunc),
			isMethod: true,
		}
	}

	for _, f := range res.Funcs {
		obj := f.Sym.Obj.(*objFunc)
		buildFunc(b, f, obj.ref.IR().(*ir.Func))
	}

	for _, f := range res.Methods {
		obj := f.Sym.Obj.(*objFunc)
		buildFunc(b, f, obj.ref.IR().(*ir.Func))
	}

	testList, testNames := buildTests(b, tops)
	addInit(b)
	addStart(b)

	if testList != nil {
		addTestStart(b, testList, len(testNames))
	}

	return tops, depGraph, testNames, nil
}
