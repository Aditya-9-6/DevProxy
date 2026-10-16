package dns

import "sync"

type QueryContext struct {
	Host string
	ID   uint16
}

var queryPool = sync.Pool{
	New: func() interface{} {
		return &QueryContext{}
	},
}

func GetQueryContext(host string, id uint16) *QueryContext {
	ctx := queryPool.Get().(*QueryContext)
	ctx.Host = host
	ctx.ID = id
	return ctx
}

func PutQueryContext(ctx *QueryContext) {
	ctx.Host = ""
	ctx.ID = 0
	queryPool.Put(ctx)
}
