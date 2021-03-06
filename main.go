package main

import (
	"image"
	"image/png"
	"log"
	"os"
	"runtime"

	"github.com/disintegration/imaging"

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
	Projection, View, InvViewProd                                      mgl32.Mat4
	Position, Target, Up, Vec00, Vec01, Vec10, Vec11                   mgl32.Vec3
	Fovy, Aspect, Near, Far                                            float32
	PosUniform, Vec00Uniform, Vec01Uniform, Vec10Uniform, Vec11Uniform int32
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

func (c *Camera) assignUniformLocations(progID uint32) {
	c.PosUniform = gl.GetUniformLocation(progID, gStr("camPos"))
	c.Vec00Uniform = gl.GetUniformLocation(progID, gStr("ray00"))
	c.Vec01Uniform = gl.GetUniformLocation(progID, gStr("ray01"))
	c.Vec10Uniform = gl.GetUniformLocation(progID, gStr("ray10"))
	c.Vec11Uniform = gl.GetUniformLocation(progID, gStr("ray11"))
}

func (c *Camera) setUniforms() {
	gl.Uniform3f(c.PosUniform, c.Position[0], c.Position[1], c.Position[2])
	gl.Uniform3f(c.Vec00Uniform, c.Vec00[0], c.Vec00[1], c.Vec00[2])
	gl.Uniform3f(c.Vec01Uniform, c.Vec01[0], c.Vec01[1], c.Vec01[2])
	gl.Uniform3f(c.Vec10Uniform, c.Vec10[0], c.Vec10[1], c.Vec10[2])
	gl.Uniform3f(c.Vec11Uniform, c.Vec11[0], c.Vec11[1], c.Vec11[2])
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

func takeScreenshot() {
	pixels := image.NewRGBA(image.Rect(0, 0, windowWidth, windowHeight))
	gl.ReadPixels(0, 0, windowWidth, windowHeight, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pixels.Pix))
	newImage := imaging.FlipV(pixels)
	output, err := os.Create("screenshot.png")
	if err != nil {
		log.Fatal(err)
	}
	png.Encode(output, newImage)
	output.Close()
}

func draw(
	vao *uint32,
	window *glfw.Window,
	computeProg *shaderutils.ShaderProgram,
	quadProg *shaderutils.ShaderProgram,
	tex *uint32,
	cam *Camera,
	count *float32) {

	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	*count++

	gl.UseProgram(computeProg.ID)

	mouseState := window.GetMouseButton(glfw.MouseButtonLeft)
	if mouseState == glfw.Press {
		mx, my := window.GetCursorPos()
		cx := (windowWidth/2 - mx) / windowWidth
		cy := (my - windowHeight/2) / windowHeight
		newCamPos := mgl32.Vec3{float32(cx), float32(cy), 1}
		newCamPos = newCamPos.Normalize()
		newCamPos = newCamPos.Mul(3)
		cam.Position = newCamPos
		cam.UpdateMatrices()
		cam.setUniforms()
		*count = float32(1.0)
	}

	gl.BindTexture(gl.TEXTURE_2D, *tex)

	countUniform := gl.GetUniformLocation(computeProg.ID, gStr("count"))

	gl.Uniform1f(countUniform, *count)

	gl.BindImageTexture(0, *tex, 0, false, 0, gl.WRITE_ONLY, gl.RGBA32F)
	gl.DispatchCompute(windowWidth/widthUnits, windowHeight/heightUnits, 1)
	gl.BindImageTexture(0, 0, 0, false, 0, gl.WRITE_ONLY, gl.RGBA32F)
	gl.MemoryBarrier(gl.SHADER_IMAGE_ACCESS_BARRIER_BIT)
	gl.BindTexture(gl.TEXTURE_2D, 0)
	gl.UseProgram(quadProg.ID)

	gl.BindVertexArray(*vao)
	gl.BindTexture(gl.TEXTURE_2D, *tex)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(quad)/3))
	gl.BindTexture(gl.TEXTURE_2D, 0)
	gl.UseProgram(0)

	window.SwapBuffers()

	if window.GetKey(glfw.KeyF3) == glfw.Press {
		takeScreenshot()
	}
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
		mgl32.Vec3{0, 0, 3},
		mgl32.Vec3{0, 0, 0},
		mgl32.Vec3{0, 1, 0},
		mgl32.DegToRad(60),
		windowWidth/windowHeight,
		0.1,
		100.0)

	vao := makeVao(quad)

	var count float32

	gl.UseProgram(computeProg.ID)
	cam.assignUniformLocations(computeProg.ID)
	cam.setUniforms()

	countUniform := gl.GetUniformLocation(computeProg.ID, gStr("count"))
	gl.Uniform1f(countUniform, count)

	comptexUniform := gl.GetUniformLocation(computeProg.ID, gStr("tex"))
	gl.Uniform1i(comptexUniform, 0)

	gl.UseProgram(quadProg.ID)
	texUniform := gl.GetUniformLocation(quadProg.ID, gStr("tex"))
	gl.Uniform1i(texUniform, 0)

	gl.UseProgram(0)

	for !window.ShouldClose() {
		draw(&vao, window, &computeProg, &quadProg, &outTex, &cam, &count)
	}
}
