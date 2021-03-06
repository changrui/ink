package render

import (
	"image"
)

// TODO is a plan resolution independent?
//      i.e. vertex data only, no texture specs.
type Plan struct {
	// TODO find a way to remove this
	//      only used for snapshots
	RootLayer int
	Shaders   map[int]Shader
	Images    map[int]image.Image
	Passes    []Pass
}

type Shader struct {
	Vert, Frag, Geom, Output string
}

type Pass struct {
	Name      string
	ShaderID  int
	Layer     int
	Vertices  int
	Instances int
	Faces     []uint32
	Uniforms  map[string]interface{}
	Attrs     map[string]Attr
}

type Attr struct {
	Value   interface{}
	Size    int
	Divisor int
}
