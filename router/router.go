package router

type Handler func(uid int64, data []byte)

type Router struct {
	handlers map[uint16]Handler
}

func NewRouter() *Router {
	return &Router{
		handlers: make(map[uint16]Handler),
	}
}

func (r *Router) Register(msgId uint16, h Handler) {
	r.handlers[msgId] = h
}

func (r *Router) Handle(uid int64, msgId uint16, data []byte) {
	if h, ok := r.handlers[msgId]; ok {
		h(uid, data)
	}
}
