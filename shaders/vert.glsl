#version 430
in vec3 vp;
out vec4 vpos;
void main() {
  gl_Position = vec4(vp, 1.0);
  vpos = vec4(vp, 1.0);
}