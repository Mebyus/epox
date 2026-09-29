package froll

type buffer struct {
	p []byte
}

func (g *buffer) put(data []byte) {
	g.p = append(g.p, data...)
}

func (g *buffer) bytes() []byte {
	return g.p
}

func (g *buffer) reset() {
	g.p = g.p[:0]
}

func (r *Roller) getBuffer() *buffer {
	return r.pool.Get().(*buffer)
}

func (w *worker) putBuffer(g *buffer) {
	g.reset()
	w.pool.Put(g)
}
