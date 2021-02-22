package main

import "C"
import (
	"fmt"
	"gopkg.in/yaml.v2"
	"image"
	"image/draw"
	_ "image/png"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

type config struct {
	ImagePath string      `yaml:"image_path"`
	Margin    int         `yaml:"margin"`
	Width     int         `yaml:"-""`
	Height    int         `yaml:"-"`
	Image     image.Image `yaml:"-"`
	Current   string      `yaml:"current"`
	Shaders   []*struct {
		Name         string `yaml:"name"`
		VertexFile   string `yaml:"vertex_file"`
		FragmentFile string `yaml:"fragment_file"`
	} `yaml:"shaders"`
}

type Shader struct {
	VertexSource   string
	FragmentSource string
	Program        uint32
}

var Config config
var Shaders map[string]*Shader

func GetShader(name string) (*Shader, error) {
	for _, _shader := range Config.Shaders {
		if _shader.Name == name {
			if _, ok := Shaders[name]; !ok {
				shader := new(Shader)

				content, err := ioutil.ReadFile(path.Join("shaders", _shader.VertexFile))
				if err != nil {
					panic(err)
				}
				shader.VertexSource = string(content) + "\x00"

				content, err = ioutil.ReadFile(path.Join("shaders", _shader.FragmentFile))
				if err != nil {
					panic(err)
				}
				shader.FragmentSource = string(content) + "\x00"

				err = shader.InitProgram()
				if err != nil {
					panic(err)
				}

				Shaders[name] = shader
			}
			return Shaders[name], nil
		}
	}
	return nil, fmt.Errorf("无法获取 %s shader", name)
}

func (s *Shader) InitProgram() error {
	vertexShader, err := compileShader(s.VertexSource, gl.VERTEX_SHADER)
	if err != nil {
		return err
	}

	fragmentShader, err := compileShader(s.FragmentSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return err
	}

	s.Program = gl.CreateProgram()

	gl.AttachShader(s.Program, vertexShader)
	gl.AttachShader(s.Program, fragmentShader)
	gl.LinkProgram(s.Program)

	var status int32
	gl.GetProgramiv(s.Program, gl.LINK_STATUS, &status)

	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(s.Program, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(s.Program, logLength, nil, gl.Str(log))

		return fmt.Errorf("failed to link program: %v", log)
	}

	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)

	return nil
}

func init() {
	// 读取配置文件
	content, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(content, &Config)
	if err != nil {
		panic(err)
	}

	img_file, err := os.Open(Config.ImagePath)
	if err != nil {
		panic(err)
	}

	Config.Image, _, err = image.Decode(img_file)
	if err != nil {
		panic(err)
	}

	Config.Width = Config.Image.Bounds().Size().X*2 + 4*Config.Margin
	Config.Height = Config.Image.Bounds().Size().Y + 2*Config.Margin
}

func compileShader(source string, shaderType uint32) (uint32, error) {
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

		return 0, fmt.Errorf("failed to compile %v: %v", source, log)
	}

	return shader, nil
}

func newTexture() (uint32, error) {

	// TODO
	rgba := image.NewRGBA(Config.Image.Bounds())
	if rgba.Stride != rgba.Rect.Size().X*4 {
		return 0, fmt.Errorf("unsupported stride")
	}
	draw.Draw(rgba, rgba.Bounds(), Config.Image, image.Point{0, 0}, draw.Src)

	var texture uint32
	gl.GenTextures(1, &texture)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(rgba.Rect.Size().X),
		int32(rgba.Rect.Size().Y),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(rgba.Pix))
	return texture, nil
}

func main() {

	if err := glfw.Init(); err != nil {
		log.Fatalln("failed to initialize glfw:", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)

	window, err := glfw.CreateWindow(Config.Width, Config.Height, "XMY", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()

	if err := gl.Init(); err != nil {
		panic(err)
	}

	//gl.UseProgram(program)

	gl.Viewport(0, 0, 320, 510)

	gl.ClearColor(0.2, 0.3, 0.3, 1.0)

	var vertexs = []float32{
		// left
		-1.0, 1.0, 0.0, 0.0, 1.0,
		-1.0, -1.0, 0.0, 0.0, 0.0,
		0.0, -1.0, 0.0, 1.0, 0.0,
		0.0, 1.0, 0.0, 1.0, 1.0,

		// right
		0.0, 1.0, 0.0, 0.0, 1.0,
		0.0, -1.0, 0.0, 0.0, 0.0,
		1.0, -1.0, 0.0, 1.0, 0.0,
		1.0, 1.0, 0.0, 1.0, 1.0,

		// others
		1.0, 1.0, 0.0, 1.0, 1.0,
		1.0, -1.0, 0.0, 1.0, 0.0,
		-1.0, -1.0, 0.0, 0.0, 0.0,
		-1.0, 1.0, 0.0, 0.0, 1.0,
	}

	fmt.Println(vertexs)

	/*

	//	textureUniform := gl.GetUniformLocation(program, gl.Str("tex\x00"))
		//	gl.Uniform1i(textureUniform, 0)

		//	gl.BindFragDataLocation(program, 0, gl.Str("outputColor\x00"))

		//	texture, err := newTexture()
		//	if err != nil {
		//		log.Fatalln(err)
		//	}
		/*
			var vao uint32
			var vbo uint32

			gl.GenVertexArrays(1, &vao)
			gl.GenBuffers(1, &vbo)

			gl.BindVertexArray(vao)
			gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
			gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)

			vertAttrib := uint32(gl.GetAttribLocation(program, gl.Str("vert\x00")))
			gl.EnableVertexAttribArray(vertAttrib)
			gl.VertexAttribPointer(vertAttrib, 3, gl.FLOAT, false, 5*4, gl.PtrOffset(0))

			texCoordAttrib := uint32(gl.GetAttribLocation(program, gl.Str("vertTexCoord\x00")))
			gl.EnableVertexAttribArray(texCoordAttrib)
			gl.VertexAttribPointer(texCoordAttrib, 2, gl.FLOAT, false, 5*4, gl.PtrOffset(3*4))

			gl.BindBuffer(gl.ARRAY_BUFFER, 0)
			gl.BindVertexArray(0)
	*/
	for !window.ShouldClose() {
		gl.Clear(gl.COLOR_BUFFER_BIT)

		//gl.BindVertexArray(vao)

		//		gl.ActiveTexture(gl.TEXTURE0)
		//		gl.BindTexture(gl.TEXTURE_2D, texture)

		//		gl.DrawArrays(gl.TRIANGLE_FAN, 0, 4)

		window.SwapBuffers()
		glfw.PollEvents()
	}

	return
}
