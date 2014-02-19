package graphics

import (
	"fmt"
	"github.com/go-gl/gl"
	glfw "github.com/go-gl/glfw3"
	"github.com/go-gl/glu"
	"runtime"
	"time"
)

const (
	vertex = `#version 330

in vec2 position;
in vec3 color;

out vec3 Color;

void main()
{
	Color = color;
    gl_Position = vec4(position, 0.0, 1.0);
}`

	fragment = `#version 330

in vec3 Color;

out vec4 outColor;

void main()
{
    outColor = vec4(Color, 1.0);
}`
)

var (
	verticies = []float32{
		0.0, 0.5, 1.0, 0.0, 0.0,
		0.5, -0.5, 0.0, 1.0, 0.0,
		-0.5, -0.5, 0.0, 0.0, 1.0}
)

type Renderer struct {
	window          *glfw.Window
	vao             gl.VertexArray
	vbo             gl.Buffer
	vertex_shader   gl.Shader
	fragment_shader gl.Shader
	program         gl.Program
	positionAttrib  gl.AttribLocation
	colorAttrib     gl.AttribLocation
}

func Init() (r *Renderer, err error) {
	r = &Renderer{}

	// lock glfw/gl calls to a single thread
	runtime.LockOSThread()

	if !glfw.Init() {
		panic("Couldn't init GLFW3")
	}

	glfw.PollEvents()

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenglForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.OpenglProfile, glfw.OpenglCoreProfile)

	r.window, err = glfw.CreateWindow(800, 600, "Example", nil, nil)
	if err != nil {
		panic(err.Error())
	}

	r.window.MakeContextCurrent()
	glfw.SwapInterval(1)

	gl.Init()

	r.vao = gl.GenVertexArray()
	r.vao.Bind()

	r.vbo = gl.GenBuffer()
	r.vbo.Bind(gl.ARRAY_BUFFER)

	gl.BufferData(gl.ARRAY_BUFFER, len(verticies)*4, verticies, gl.STATIC_DRAW)

	r.vertex_shader = gl.CreateShader(gl.VERTEX_SHADER)
	r.vertex_shader.Source(vertex)
	r.vertex_shader.Compile()
	fmt.Println(r.vertex_shader.GetInfoLog())

	r.fragment_shader = gl.CreateShader(gl.FRAGMENT_SHADER)
	r.fragment_shader.Source(fragment)
	r.fragment_shader.Compile()
	fmt.Println(r.fragment_shader.GetInfoLog())

	r.program = gl.CreateProgram()
	r.program.AttachShader(r.vertex_shader)
	r.program.AttachShader(r.fragment_shader)

	r.program.BindFragDataLocation(0, "outColor")
	r.program.Link()
	r.program.Use()

	r.positionAttrib = r.program.GetAttribLocation("position")
	r.positionAttrib.EnableArray()
	r.positionAttrib.AttribPointer(2, gl.FLOAT, false, 5*4, nil)

	r.colorAttrib = r.program.GetAttribLocation("color")
	r.colorAttrib.EnableArray()
	r.colorAttrib.AttribPointer(3, gl.FLOAT, false, 2*4, nil)

	gl.ClearColor(0.2, 0.2, 0.23, 0.0)

	// open the window draw 10 frames, hopefully glfw settles down a little
	// without this, it doesn't draw the window decoration, catch keypresses etc.
	for i := 0; i < 10; i++ {
		glfw.PollEvents()
		r.DrawFrame()
		time.Sleep(10 * time.Millisecond)
	}

	return
}

func (r *Renderer) Shutdown() {
	r.colorAttrib.DisableArray()
	r.colorAttrib = 0

	r.positionAttrib.DisableArray()
	r.positionAttrib = 0

	r.program.Delete()
	r.program = 0

	r.fragment_shader.Delete()
	r.fragment_shader = 0

	r.vertex_shader.Delete()
	r.vertex_shader = 0

	r.vbo.Delete()
	r.vbo = 0

	r.vao.Delete()
	r.vao = 0

	r.window.Destroy()
}

func (r *Renderer) DrawFrame() {
	gl.Clear(gl.COLOR_BUFFER_BIT)
	gl.DrawArrays(gl.TRIANGLES, 0, 3)

	r.window.SwapBuffers()
	glfw.PollEvents()

	if r.window.GetKey(glfw.KeyEscape) == glfw.Press {
		r.window.SetShouldClose(true)
	}
}

func (r *Renderer) ShouldClose() bool {
	return r.window.ShouldClose()
}

func checkGLerror() {
	if glerr := gl.GetError(); glerr != gl.NO_ERROR {
		string, _ := glu.ErrorString(glerr)
		panic(string)
	}
}
