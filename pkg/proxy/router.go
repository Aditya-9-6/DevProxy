package proxy

import (
	"github.com/Aditya-9-6/DevProxy/pkg/router"
)

type ProxyRouter struct {
	engine *router.RadixRouter
}

func NewProxyRouter() *ProxyRouter {
	return &ProxyRouter{engine: router.NewRadixRouter()}
}

func (pr *ProxyRouter) Register(path string, handler interface{}) {
	pr.engine.Add(path, handler)
}

func (pr *ProxyRouter) Route(path string) interface{} {
	return pr.engine.Lookup(path)
}
