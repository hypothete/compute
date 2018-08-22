package main

import (
	"log"
	"runtime"

	"github.com/go-gl/gl/v4.3-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/hypothete/compute/lib"
)

const (
	windowWidth  = 1024
	windowHeight = 512
	widthUnits   = 16
	heightUnits  = 8
)

var (
	quad = []float32{
		-1, -1, 0, // top
		1, -1, 0, // left
		-1, 1, 0, // right
		1, -1, 0, // top
		-1, 1, 0, // left
		1, 1, 0, // right
	}
)

// Camera is used to view the scene
type Camera struct {
	Projection, View, InvViewProd                    mgl32.Mat4
	Position, Target, Up, Vec00, Vec01, Vec10, Vec11 mgl32.Vec3
	Fovy, Aspect, Near, Far                          float32
}

// UpdateMatrices reads values form the camera and preps the matrices
func (c *Camera) UpdateMatrices() {
	c.View = mgl32.LookAtV(c.Position, c.Target, c.Up)
	c.Projection = mgl32.Perspective(c.Fovy, c.Aspect, c.Near, c.Far)
	c.InvViewProd = c.Projection.Mul4(c.View)
	c.InvViewProd = c.InvViewProd.Inv()
	vec00 := mgl32.Vec4{-1, -1, 0, 1}
	vec01 := mgl32.Vec4{-1, 1, 0, 1}
	vec10 := mgl32.Vec4{1, -1, 0, 1}
	vec11 := mgl32.Vec4{1, 1, 0, 1}

	vec00 = c.InvViewProd.Mul4x1(vec00)
	vec01 = c.InvViewProd.Mul4x1(vec01)
	vec10 = c.InvViewProd.Mul4x1(vec10)
	vec11 = c.InvViewProd.Mul4x1(vec11)

	vec00 = vec00.Mul(1 / vec00.W())
	vec01 = vec01.Mul(1 / vec01.W())
	vec10 = vec10.Mul(1 / vec10.W())
	vec11 = vec11.Mul(1 / vec11.W())

	vec00 = vec00.Sub(c.Position.Vec4(0))
	vec01 = vec01.Sub(c.Position.Vec4(0))
	vec10 = vec10.Sub(c.Position.Vec4(0))
	vec11 = vec11.Sub(c.Position.Vec4(0))

	c.Vec00 = vec00.Vec3()
	c.Vec01 = vec01.Vec3()
	c.Vec10 = vec10.Vec3()
	c.Vec11 = vec11.Vec3()
}

// NewCamera is a camera constructor
func NewCamera(position, target, up mgl32.Vec3, fovy, aspect, near, far float32) Camera {
	c := new(Camera)
	c.Position = position
	c.Target = target
	c.Up = up
	c.Fovy = fovy
	c.Aspect = aspect
	c.Near = near
	c.Far = far
	c.UpdateMatrices()
	return *c
}

// gStr is a shorthand for goofy string concat
func gStr(str string) *uint8 {
	formatted := gl.Str(str + "\x00")
	return formatted
}

// initGlfw initializes glfw and returns a Window to use.
func initGlfw() *glfw.Window {
	if err := glfw.Init(); err != nil {
		panic(err)
	}

	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	window, err := glfw.CreateWindow(windowWidth, windowHeight, "Compute", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()

	return window
}

// makeVao initializes and returns a vertex array from the points provided.
func makeVao(points []float32) uint32 {
	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(points), gl.Ptr(points), gl.STATIC_DRAW)

	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)
	gl.EnableVertexAttribArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 0, nil)
	return vao
}

func makeOutputTexture(renderedTexture uint32) uint32 {
	gl.GenTextures(1, &renderedTexture)
	gl.BindTexture(gl.TEXTURE_2D, renderedTexture)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA32F, windowWidth, windowHeight, 0, gl.RGBA, gl.FLOAT, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.BindTexture(gl.TEXTURE_2D, 0)
	return renderedTexture
}

func draw(
	vao uint32,
	window *glfw.Window,
	computeProg shaderutils.ShaderProgram,
	quadProg shaderutils.ShaderProgram,
	tex uint32,
	cam Camera) {

	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	mx, my := window.GetCursorPos()
	cx := (windowWidth/2 - mx) / windowWidth
	cy := (my - windowHeight/2) / windowHeight
	newCamPos := mgl32.Vec3{float32(cx), float32(cy), 1}
	newCamPos = newCamPos.Normalize()
	newCamPos = newCamPos.Mul(5)
	cam.Position = newCamPos
	cam.UpdateMatrices()

	gl.UseProgram(computeProg.ID)

	camPosUniform := gl.GetUniformLocation(computeProg.ID, gStr("camPos"))
	vec00Uniform := gl.GetUniformLocation(computeProg.ID, gStr("ray00"))
	vec01Uniform := gl.GetUniformLocation(computeProg.ID, gStr("ray01"))
	vec10Uniform := gl.GetUniformLocation(computeProg.ID, gStr("ray10"))
	vec11Uniform := gl.GetUniformLocation(computeProg.ID, gStr("ray11"))

	gl.Uniform3f(camPosUniform, cam.Position[0], cam.Position[1], cam.Position[2])

	gl.Uniform3f(vec00Uniform, cam.Vec00[0], cam.Vec00[1], cam.Vec00[2])
	gl.Uniform3f(vec01Uniform, cam.Vec01[0], cam.Vec01[1], cam.Vec01[2])
	gl.Uniform3f(vec10Uniform, cam.Vec10[0], cam.Vec10[1], cam.Vec10[2])
	gl.Uniform3f(vec11Uniform, cam.Vec11[0], cam.Vec11[1], cam.Vec11[2])

	gl.BindImageTexture(0, tex, 0, false, 0, gl.WRITE_ONLY, gl.RGBA32F)
	gl.DispatchCompute(windowWidth/widthUnits, windowHeight/heightUnits, 1)
	gl.BindImageTexture(0, 0, 0, false, 0, gl.WRITE_ONLY, gl.RGBA32F)
	gl.MemoryBarrier(gl.SHADER_IMAGE_ACCESS_BARRIER_BIT)

	gl.UseProgram(quadProg.ID)

	gl.BindVertexArray(vao)
	gl.BindTexture(gl.TEXTURE_2D, tex)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(quad)/3))
	gl.BindTexture(gl.TEXTURE_2D, 0)
	gl.UseProgram(0)

	window.SwapBuffers()
	glfw.PollEvents()
}

// initOpenGL initializes OpenGL
func initOpenGL() {
	if err := gl.Init(); err != nil {
		panic(err)
	}
	version := gl.GoStr(gl.GetString(gl.VERSION))
	log.Println("OpenGL version", version)
}

func main() {
	runtime.LockOSThread()

	window := initGlfw()
	defer glfw.Terminate()

	initOpenGL()

	computeShader := shaderutils.Load("shaders/compute.glsl", gl.COMPUTE_SHADER)
	vertexShader := shaderutils.Load("shaders/vert.glsl", gl.VERTEX_SHADER)
	fragmentShader := shaderutils.Load("shaders/frag.glsl", gl.FRAGMENT_SHADER)

	computeProg := shaderutils.CreateShaderProgram()
	computeProg.Attach(computeShader)
	computeProg.Link()

	quadProg := shaderutils.CreateShaderProgram()
	quadProg.Attach(vertexShader)
	quadProg.Attach(fragmentShader)
	quadProg.Link()

	outTex := makeOutputTexture(42)

	cam := NewCamera(
		mgl32.Vec3{3, 2, 7},
		mgl32.Vec3{0, 0.5, 0},
		mgl32.Vec3{0, 1, 0},
		mgl32.DegToRad(60),
		windowWidth/windowHeight,
		0.1,
		100.0)

	vao := makeVao(quad)

	gl.UseProgram(computeProg.ID)
	camPosUniform := gl.GetUniformLocation(computeProg.ID, gStr("camPos"))
	vec00Uniform := gl.GetUniformLocation(computeProg.ID, gStr("ray00"))
	vec01Uniform := gl.GetUniformLocation(computeProg.ID, gStr("ray01"))
	vec10Uniform := gl.GetUniformLocation(computeProg.ID, gStr("ray10"))
	vec11Uniform := gl.GetUniformLocation(computeProg.ID, gStr("ray11"))

	gl.Uniform3f(camPosUniform, cam.Position[0], cam.Position[1], cam.Position[2])

	gl.Uniform3f(vec00Uniform, cam.Vec00[0], cam.Vec00[1], cam.Vec00[2])
	gl.Uniform3f(vec01Uniform, cam.Vec01[0], cam.Vec01[1], cam.Vec01[2])
	gl.Uniform3f(vec10Uniform, cam.Vec10[0], cam.Vec10[1], cam.Vec10[2])
	gl.Uniform3f(vec11Uniform, cam.Vec11[0], cam.Vec11[1], cam.Vec11[2])

	gl.UseProgram(quadProg.ID)
	texUniform := gl.GetUniformLocation(quadProg.ID, gStr("tex"))
	gl.Uniform1i(texUniform, 0)

	gl.UseProgram(0)

	for !window.ShouldClose() {
		draw(vao, window, computeProg, quadProg, outTex, cam)
	}
}
