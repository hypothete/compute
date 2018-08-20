#version 430
in vec4 vpos;
uniform sampler2D tex;
out vec4 frag_color;
void main() {
  vec3 texColor = texture(tex, vpos.xy / 2.0 + vec2(0.5)).rgb;
  frag_color = vec4(texColor, 1.0);
}