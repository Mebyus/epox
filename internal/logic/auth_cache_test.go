package logic

import (
	"slices"
	"testing"
)

func Test_cubuf_cull(t *testing.T) {
	tests := []struct {
		name string
		x    uint32 // cull value
		p    uint32 // initial buffer pos
		init []uint32
		want []uint32
	}{
		{
			name: "1 empty",
			x:    0,
			p:    0,
			init: nil,
			want: nil,
		},
		{
			name: "2 at start",
			x:    6,
			p:    0,
			init: []uint32{3, 4, 6, 7, 10, 1, 2},
			want: []uint32{3, 4, 1, 2},
		},
		{
			name: "2 at end",
			x:    6,
			p:    scap - 3,
			init: []uint32{3, 4, 6, 7, 10, 1, 2},
			want: []uint32{3, 4, 1, 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := cubuf{p: tt.p}
			for _, a := range tt.init {
				c.push(a)
			}
			c.cull(tt.x)

			got := c.list()
			if !slices.Equal(got, tt.want) {
				t.Errorf("list() = %v, want %v", got, tt.want)
			}
		})
	}
}
