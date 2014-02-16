package gophercraft

import (
	"encoding/binary"
	"fmt"
	"runtime"
)

import (
	"github.com/go-gl/gl"
	glfw "github.com/go-gl/glfw3"
	"github.com/go-gl/glu"
)

const (
	vertex = `#version 330

in vec2 position;

void main()
{
    gl_Position = vec4(position, 0.0, 1.0);
}`

	fragment = `#version 330

out vec4 outColor;

void main()
{
    outColor = vec4(1.0, 1.0, 1.0, 1.0);
}`
)

var (
	varintBuff [binary.MaxVarintLen64]byte
)

func main() {
	/// lock glfw/gl calls to a single thread
	runtime.LockOSThread()

	glfw.Init()
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenglForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.OpenglProfile, glfw.OpenglCoreProfile)

	window, err := glfw.CreateWindow(800, 600, "Example", nil, nil)
	if err != nil {
		panic(err)
	}

	defer window.Destroy()

	window.MakeContextCurrent()
	glfw.SwapInterval(1)

	gl.Init()

	vao := gl.GenVertexArray()
	vao.Bind()

	vbo := gl.GenBuffer()
	vbo.Bind(gl.ARRAY_BUFFER)

	verticies := []float32{0, 1, 0, -1, -1, 0, 1, -1, 0}

	gl.BufferData(gl.ARRAY_BUFFER, len(verticies)*4, verticies, gl.STATIC_DRAW)

	vertex_shader := gl.CreateShader(gl.VERTEX_SHADER)
	vertex_shader.Source(vertex)
	vertex_shader.Compile()
	fmt.Println(vertex_shader.GetInfoLog())
	defer vertex_shader.Delete()

	fragment_shader := gl.CreateShader(gl.FRAGMENT_SHADER)
	fragment_shader.Source(fragment)
	fragment_shader.Compile()
	fmt.Println(fragment_shader.GetInfoLog())
	defer fragment_shader.Delete()

	program := gl.CreateProgram()
	program.AttachShader(vertex_shader)
	program.AttachShader(fragment_shader)

	program.BindFragDataLocation(0, "outColor")
	program.Link()
	program.Use()
	defer program.Delete()

	positionAttrib := program.GetAttribLocation("position")
	positionAttrib.AttribPointer(3, gl.FLOAT, false, 0, nil)
	positionAttrib.EnableArray()
	defer positionAttrib.DisableArray()

	host := "localhost"
	port := 25565

	json := pingServer(host, port)
	fmt.Printf("Ping Response:\n%s\n\n", json)

	joinServer(host, port, window)
}

func checkGLerror() {
	if glerr := gl.GetError(); glerr != gl.NO_ERROR {
		string, _ := glu.ErrorString(glerr)
		panic(string)
	}
}

func frame(window *glfw.Window) {
	gl.ClearColor(0.2, 0.2, 0.23, 0.0)
	gl.Clear(gl.COLOR_BUFFER_BIT)

	gl.DrawArrays(gl.TRIANGLES, 0, 3)

	window.SwapBuffers()
	glfw.PollEvents()

}

func checkError(err error) {
	if err != nil {
		panic(err.Error())
	}
}
