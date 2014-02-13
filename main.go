package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	// "math"
	"net"
	// "os"
	// "strings"
	// "encoding/hex"
	"runtime"
	"strconv"
	"time"
)

import (
	"github.com/go-gl/gl"
	glfw "github.com/go-gl/glfw3"
	"github.com/go-gl/glu"
)

var (
	varintBuff [binary.MaxVarintLen64]byte
)

func main() {
	// needed for glfw3 not to seg at the moment
	runtime.LockOSThread()

	fmt.Println("Gophercraft!\n")

	if !glfw.Init() {
		panic("Can't init glfw3!")
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenglForwardCompatible, gl.TRUE)
	glfw.WindowHint(glfw.OpenglProfile, glfw.OpenglCoreProfile)

	major, minor, revision := glfw.GetVersion()
	fmt.Printf("OpenGL Version: %d %d %d\n", major, minor, revision)

	window, err := glfw.CreateWindow(800, 600, "Gophercraft!", nil, nil)

	if err != nil {
		panic(err)
	}

	window.MakeContextCurrent()
	glfw.SwapInterval(1)

	gl.Init()

	window.SwapBuffers()
	defer window.Destroy()

	checkGLerror()

	vertex := `#version 150

in vec2 position;

void main()
{
    gl_Position = vec4(position, 0.0, 1.0);
}`

	vertex_shader := gl.CreateShader(gl.VERTEX_SHADER)
	vertex_shader.Source(vertex)
	vertex_shader.Compile()
	fmt.Println(vertex_shader.GetInfoLog())

	checkGLerror()

	fragment := `#version 150

out vec4 outColor;

void main()
{
    outColor = vec4(1.0, 1.0, 1.0, 1.0);
}`

	fragment_shader := gl.CreateShader(gl.FRAGMENT_SHADER)
	fragment_shader.Source(fragment)
	fragment_shader.Compile()
	fmt.Println(fragment_shader.GetInfoLog())
	checkGLerror()

	program := gl.CreateProgram()
	program.AttachShader(vertex_shader)
	program.AttachShader(fragment_shader)

	program.BindFragDataLocation(0, "outColor")
	program.Link()
	program.Use()

	positionAttrib := program.GetAttribLocation("position")
	positionAttrib.AttribPointer(2, gl.FLOAT, false, 0, nil)
	positionAttrib.EnableArray()

	checkGLerror()

	var vbo = gl.GenBuffer()
	vbo.Bind(gl.ARRAY_BUFFER)

	verticies := []float32{
		0.0, 0.5,
		0.5, -0.5,
		-0.5, -0.5}

	gl.BufferData(gl.GLenum(vbo), 24, verticies, gl.STATIC_DRAW)

	checkGLerror()

	vertex_array := gl.GenVertexArray()
	vertex_array.Bind()

	checkGLerror()

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

func connect(host string, port int) net.Conn {
	conn, err := net.Dial("tcp", host+":"+strconv.Itoa(port))
	checkError(err)

	return conn
}

func pingServer(host string, port int) string {
	conn := connect(host, port)
	defer conn.Close()

	sendHandshake(conn, host, port, 1)
	sendStatusRequest(conn)
	return readStatusRequest(conn)
}

type Packet struct {
	id   int
	size int
	data []byte
}

func joinServer(host string, port int, window *glfw.Window) {
	conn := connect(host, port)
	defer conn.Close()

	fmt.Println("Going to join server")

	sendHandshake(conn, host, port, 2)
	sendLoginStart(conn)

	tick := make(chan bool)

	go func() {
		for {
			time.Sleep(40 * time.Millisecond)
			tick <- true
		}
	}()

	reader := bufio.NewReader(conn)
	readLoginSuccess(reader)

	fromServer := make(chan Packet)

	go func() {
		reader := bufio.NewReader(conn)

		for {
			size, err := binary.ReadUvarint(reader)
			if err != nil {
				fmt.Println("Error reading from connection, returning from reciver. Error: " + err.Error())
				return
			}

			// ignore the packet type in the packet length since we read it too
			size -= 1

			packetType, err := binary.ReadUvarint(reader)
			checkError(err)

			data := make([]byte, size)
			n, _ := reader.Read(data)

			for n < int(size) {
				// didn't read all the data, keep trying to read some more
				nx, _ := reader.Read(data[n:])
				n += nx
			}

			var packet Packet
			packet.id = int(packetType)
			packet.size = int(size)
			packet.data = data

			fromServer <- packet
		}
	}()

	tickCount := 0

	for {
		select {
		case <-tick:
			tickCount += 1
			fmt.Printf("Tick %d\n", tickCount)

			frame(window)

			if tickCount >= 200 {
				return
			}
		case packet := <-fromServer:
			fmt.Printf("Packet: 0x%0x\t\tsize: %d\n", packet.id, packet.size)

			switch packet.id {
			case 0:
				fmt.Printf("Keep alive: %d\n", packet.data)

				databuffer := bytes.NewBuffer(packet.data)
				var keepalive int
				binary.Read(databuffer, binary.BigEndian, &keepalive)

				fmt.Printf("Keepalive was: %d\n", keepalive)

				// send it right back
				buf := new(bytes.Buffer)

				writeVarint(buf, 0)         // packet id
				writeData(buf, packet.data) // keep alive id

				sendPacket(conn, buf)
			}
		}
	}
}

func frame(window *glfw.Window) {
	gl.ClearColor(0.2, 0.2, 0.23, 0.0)
	gl.Clear(gl.COLOR_BUFFER_BIT)

	gl.DrawArrays(gl.TRIANGLES, 0, 3)

	window.SwapBuffers()
	glfw.PollEvents()

}

func sendHandshake(conn net.Conn, host string, port int, state int) {
	buf := new(bytes.Buffer)

	writeVarint(buf, 0)            // packet id
	writeVarint(buf, 4)            // protocol version
	writeString(buf, host)         // server address
	writeData(buf, uint16(port))   // server port
	writeVarint(buf, int64(state)) // next state 1 for status, 2 for login

	sendPacket(conn, buf)
}

func sendStatusRequest(conn net.Conn) {
	buf := new(bytes.Buffer)

	writeVarint(buf, 0) // packet id

	sendPacket(conn, buf)
}

func sendLoginStart(conn net.Conn) {
	buf := new(bytes.Buffer)

	writeVarint(buf, 0)             // packet id
	writeString(buf, "gophercraft") // username

	sendPacket(conn, buf)
}

func readStatusRequest(conn net.Conn) string {
	reader := bufio.NewReader(conn)

	_, err := binary.ReadUvarint(reader)
	checkError(err)

	_, err = binary.ReadUvarint(reader)
	checkError(err)

	return readString(reader)
}

func readLoginSuccess(reader *bufio.Reader) {
	var err error

	_, err = binary.ReadUvarint(reader)
	checkError(err)

	if id, _ := binary.ReadUvarint(reader); id != 2 {
		panic("Expected packet id 2 (Login Success), got: " + strconv.Itoa(int(id)))
	}

	fmt.Println("Got login success packet!")

	uuid := readString(reader)
	fmt.Printf("UUID: %s\n", uuid)

	username := readString(reader)
	fmt.Printf("Username: %s\n", username)
}

func checkError(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func writeVarint(w io.Writer, varint int64) error {
	n := binary.PutUvarint(varintBuff[:], uint64(varint))
	_, err := w.Write(varintBuff[0:n])

	checkError(err)

	return err
}

func writeString(w io.Writer, s string) (err error) {
	bs := []byte(s)
	writeVarint(w, int64(len(bs)))

	_, err = w.Write(bs)
	checkError(err)

	return
}

func readString(reader *bufio.Reader) string {
	stringSize, err := binary.ReadUvarint(reader)
	checkError(err)

	stringbytes := make([]byte, stringSize)
	reader.Read(stringbytes)

	return string(stringbytes)
}

func writeData(w io.Writer, data interface{}) {
	err := binary.Write(w, binary.BigEndian, data)
	checkError(err)
}

func sendPacket(w io.Writer, buf *bytes.Buffer) {
	sendBuffer := bufio.NewWriter(w)

	writeVarint(sendBuffer, int64(len(buf.Bytes())))
	buf.WriteTo(sendBuffer)

	sendBuffer.Flush()
}
