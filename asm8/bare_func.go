package asm8

import (
	"io"

	"e8vm.io/e8vm/asm8/parse"
	"e8vm.io/e8vm/lex8"
	"e8vm.io/e8vm/link8"
)

// BuildBareFunc builds a function body into an image.
func BuildBareFunc(f string, rc io.ReadCloser) ([]byte, []*lex8.Error) {
	fn, es := parse.BareFunc(f, rc)
	if es != nil {
		return nil, es
	}

	// resolving pass
	log := lex8.NewErrorList()
	rfunc := resolveFunc(log, fn)
	if es := log.Errs(); es != nil {
		return nil, es
	}

	// building pass
	b := newBuilder("main")
	fobj := buildFunc(b, rfunc)
	if es := b.Errs(); es != nil {
		return nil, es
	}

	ret, e := link8.LinkBareFunc(fobj)
	if e != nil {
		return nil, lex8.SingleErr(e)
	}

	return ret, nil
}
