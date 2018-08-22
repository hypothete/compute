#version 430 core
layout(binding = 0, rgba32f) uniform image2D framebuffer;
layout (local_size_x = 16, local_size_y = 8) in;

#define MAX_SCENE_BOUNDS 100.0
#define NUM_BOXES 2
#define NUM_SPHERES 3

// The camera specification
uniform vec3 camPos;
uniform vec3 ray00;
uniform vec3 ray01;
uniform vec3 ray10;
uniform vec3 ray11;

struct box {
  vec3 min;
  vec3 max;
};

struct sphere {
  vec3 center;
  float radius;
};

struct hitinfo {
  vec2 lambda;
  int index;
};

struct ray {
  vec3 origin;
  vec3 dir;
};

const box boxes[] = {
  /* The ground */
  {vec3(-5.0, -0.1, -5.0), vec3(5.0, 0.0, 5.0)},
  /* Box in the middle */
  {vec3(-0.5, 0.0, -0.5), vec3(0.5, 1.0, 0.5)}
};

const sphere spheres[] = {
  {vec3(-2.0, 0.0, 0.0), 0.5},
  {vec3(2.0, 0.0, 2.0), 0.25},
  {vec3(1.0, 0.0, -0.5), 0.75}
};

const vec3 colors[] = {
  vec3(1.0, 0.0, 0.0),
  vec3(0.0, 1.0, 0.0),
  vec3(0.0, 0.0, 1.0)
};

vec2 intersectBox(ray r, const box b) {
  vec3 tMin = (b.min - r.origin) / r.dir;
  vec3 tMax = (b.max - r.origin) / r.dir;
  vec3 t1 = min(tMin, tMax);
  vec3 t2 = max(tMin, tMax);
  float tNear = max(max(t1.x, t1.y), t1.z);
  float tFar = min(min(t2.x, t2.y), t2.z);
  return vec2(tNear, tFar);
}

bool intersectBoxes(ray r, out hitinfo info) {
  float smallest = MAX_SCENE_BOUNDS;
  bool found = false;
  for (int i = 0; i < NUM_BOXES; i++) {
    vec2 lambda = intersectBox(r, boxes[i]);
    if (lambda.x > 0.0 && lambda.x < lambda.y && lambda.x < smallest) {
      info.lambda = lambda;
      info.index = i;
      smallest = lambda.x;
      found = true;
    }
  }
  return found;
}

bool intersectSphere(ray r, const sphere s) {
  vec3 oc = r.origin - s.center;
  float a = dot(r.dir, r.dir);
  float b = 2.0 * dot(oc, r.dir);
  float c = dot(oc, oc) - s.radius * s.radius;
  float discriminant = b*b - 4.0 * a * c;
  return discriminant > 0.0;
}

bool intersectSpheres(ray r, out hitinfo info) {
  bool found = false;
  float smallest = MAX_SCENE_BOUNDS;
  for (int i = 0; i < NUM_SPHERES; i++) {
    bool hitSphere = intersectSphere(r, spheres[i]);
    if (hitSphere) {
      found = true;
      info.lambda = vec2(0.0);
      info.index = i;
    }
  }
  return found;
}


vec4 trace(ray r) {
  hitinfo i;
  // if (intersectBoxes(r, i)) {
  //   vec4 gray = vec4(i.index / 10.0 + 0.8);
  //   return vec4(gray.rgb, 1.0);
  // }
  if (intersectSpheres(r, i)) {
    return vec4(colors[i.index], 1.0);
  }
  return vec4(0.0, 0.0, 0.0, 1.0);
}

void main(void) {
  ivec2 pix = ivec2(gl_GlobalInvocationID.xy);
  ivec2 size = imageSize(framebuffer);
  if (pix.x >= size.x || pix.y >= size.y) {
    return;
  }

  vec2 screen = vec2(pix) / vec2(size.x, size.y);
  ray r;
  r.origin = camPos;
  r.dir = mix(mix(ray00.xyz, ray01.xyz, screen.y), mix(ray10.xyz, ray11.xyz, screen.y), screen.x);
  vec4 color = trace(r);
  imageStore(framebuffer, pix, color);
}