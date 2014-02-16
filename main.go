package main

import (
	"encoding/binary"
	"fmt"
	"runtime"
	"time"
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
	host       = "localhost"
	port       = 25565
)

func main() {
	fmt.Println("Gophercraft!\n")

	// ping the minecraft server to see if it is there before moving further
	json, err := Ping(host, port)
	fmt.Printf("Ping Response:\n%s\n", json)

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

	gl.ClearColor(0.2, 0.2, 0.23, 0.0)

	client, err := JoinServer(host, port)
	if err != nil {
		panic(err)
	}

	defer client.Shutdown()

	run(window, client)
}

func checkGLerror() {
	if glerr := gl.GetError(); glerr != gl.NO_ERROR {
		string, _ := glu.ErrorString(glerr)
		panic(string)
	}
}

func run(window *glfw.Window, client *Client) {
	// start the tick goroutine
	tick := make(chan bool)
	go tick_run(20, tick)

	// open the window draw a frame
	frame(window)

	for !window.ShouldClose() {
		select {
		case <-tick:
			frame(window)
		case packet := <-client.packets:
			handlePacket(client, packet)
		}
	}

}

func tick_run(everyMS int, tick chan bool) {
	sleepTime := time.Duration(everyMS) * time.Millisecond

	for {
		time.Sleep(sleepTime)
		tick <- true
	}
}

func frame(window *glfw.Window) {
	gl.Clear(gl.COLOR_BUFFER_BIT)
	gl.DrawArrays(gl.TRIANGLES, 0, 3)

	window.SwapBuffers()
	glfw.PollEvents()

	if window.GetKey(glfw.KeyEscape) == glfw.Press {
		window.SetShouldClose(true)
	}
}

func handlePacket(client *Client, packet *Packet) {
	switch packet.Id {
	case 0x0:
		keepalive, err := ParseKeepalive(packet)

		if err != nil {
			panic(err)
		}

		fmt.Printf("Keepalive was: %d\n", keepalive)

		client.SendKeepAlive(keepalive)
	case 0x26:
		err := ParseMapChunkBulk(packet)

		if err != nil {
			panic(err)
		}

	default:
		fmt.Printf("Packet: 0x%0x\tsize: %d\tnot handled\n", packet.Id, packet.Size)
	}
}

func checkError(err error) {
	if err != nil {
		panic(err.Error())
	}
}
