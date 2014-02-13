package main

import (
	_ "fmt"
	"runtime"
)

import (
	"github.com/go-gl/gl"
	_ "github.com/go-gl/glu"
	"github.com/jackyb/go-sdl2/sdl"
)

func main() {
	// lock sdl calls to the main thread
	runtime.LockOSThread()

	sdl.Init(sdl.INIT_VIDEO)

	sdl.GL_SetAttribute(sdl.GL_CONTEXT_MAJOR_VERSION, 3)
	sdl.GL_SetAttribute(sdl.GL_CONTEXT_MINOR_VERSION, 3)
	sdl.GL_SetAttribute(sdl.GL_CONTEXT_FORWARD_COMPATIBLE_FLAG, gl.TRUE)
	sdl.GL_SetAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_CORE)

	window := sdl.CreateWindow("GopherCraft!", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, 800, 600, sdl.WINDOW_OPENGL|sdl.WINDOW_SHOWN|sdl.WINDOW_RESIZABLE)
	defer window.Destroy()

	// Create an OpenGL context associated with the window.
	glContext := sdl.GL_CreateContext(window)
	defer sdl.GL_DeleteContext(glContext)

	gl.Init()

	// // now you can make GL calls.
	gl.ClearColor(0, 0, 0, 1)
	// gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	// sdl.GL_SwapWindow(window)

	sdl.Delay(1000)
}
