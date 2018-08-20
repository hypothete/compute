package shaderutils

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/go-gl/gl/v4.3-core/gl"
)

// ShaderProgram is a struct describing a set of shaders compiled into a program
type ShaderProgram struct {
	ID uint32
}

// CreateShaderProgram creates the gl reference for the ShaderProgram
func CreateShaderProgram() ShaderProgram {
	var sp = ShaderProgram{ID: gl.CreateProgram()}
	return sp
}

// Attach is a wrapper for gl.AttachShader
func (sp ShaderProgram) Attach(shaderID uint32) {
	gl.AttachShader(sp.ID, shaderID)
}

// Link is a wrapper for gl.LinkProgram
func (sp ShaderProgram) Link() {
	gl.LinkProgram(sp.ID)
}

// Load takes a file path and tries to load it as a shader
func Load(path string, shaderType uint32) uint32 {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	source := string(bytes) + "\x00"

	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		panic(fmt.Errorf("failed to compile %v: %v", source, log))
	}

	return shader
}
