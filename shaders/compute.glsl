#version 430 core
layout(binding = 0, rgba32f) uniform image2D framebuffer;
layout (local_size_x = 16, local_size_y = 8) in;

#define MAX_SCENE_BOUNDS 100.0
#define NUM_BOXES 2
#define NUM_SPHERES 3
#define EPSILON 0.0001
#define EXPOSURE 8.0

// The camera specification
uniform vec3 camPos;
uniform vec3 ray00;
uniform vec3 ray01;
uniform vec3 ray10;
uniform vec3 ray11;
uniform float count;
uniform sampler2D tex;

struct box {
  vec3 min;
  vec3 max;
  int color;
};

struct sphere {
  vec3 center;
  float radius;
  int color;
};

struct hitinfo {
  vec2 lambda;
  int index;
  vec3 normal;
};

struct ray {
  vec3 origin;
  vec3 dir;
};

struct material {
  vec3 diffuse;
  float specular;
  vec3 emissive;
};

const box boxes[] = {
  /* The ground */
  {vec3(-5.0, -1.0, -5.0), vec3(5.0, -0.75, 5.0), 0},
  /* Box in the middle */
  {vec3(-1.0, -0.75, -0.5), vec3(0.0, 0.25, 0.5), 1}
};

const sphere spheres[] = {
  {vec3(0.0, 3.0, 0.0), 1.0, 2}, // light
  {vec3(-0.5, 0.5, 0.0), 0.25, 3}, // pink
  {vec3(1.25, 0.0, -0.25), 0.75, 4} // yellow
};

const material materials[] = {
  { vec3(0.7, 0.7, 0.9), 0.0, vec3(0.0, 0.0, 0.0) },
  { vec3(0.1, 0.4, 0.1), 0.0, vec3(0.0, 0.0, 0.0) },
  { vec3(1.0, 0.9, 0.8), 0.0, vec3(1.0, 0.9, 0.8) },
  { vec3(1.0, 0.5, 0.8), 0.0, vec3(0.0, 0.0, 0.0) },
  { vec3(1.0, 0.9, 0.5), 1.0, vec3(0.0, 0.0, 0.0) }
};

float rand(vec2 co) {
  return fract(sin(dot(co.xy ,vec2(12.9898,78.233))) * 43758.5453);
}

vec3 randomOnUnitSphere(vec2 q) {
  vec3 p;

  float x = rand(q * vec2(-1.0, 7.0));
  float y = rand(q * vec2(9.0, 3.0));
  float z = rand(q * vec2(-22.0, 4.0));
  x = x / cos(x);
  y = y / cos(y);
  z = z / cos(z);
  p = 2.0 * vec3(x,y,z) - 1.0;
  p = normalize(p);
  return p;
}

vec3 pointAt(ray r, float t) {
  return r.origin + r.dir * t;
}

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
  float smallest = info.lambda.x;
  bool found = false;
  for (int i = 0; i < NUM_BOXES; i++) {
    vec2 lambda = intersectBox(r, boxes[i]);
    if (lambda.x > 0.0 && lambda.x < lambda.y && lambda.x < smallest) {
      info.lambda = lambda;
      info.index = boxes[i].color;
      smallest = lambda.x;
      found = true;
      vec3 pt1 = pointAt(r, lambda.x);
      if (abs(pt1.x - boxes[i].max.x) < EPSILON) {
        info.normal = vec3(1.0, 0.0, 0.0);
      }
      else if (abs(pt1.x - boxes[i].min.x) < EPSILON) {
        info.normal = vec3(-1.0, 0.0, 0.0);
      }
      else if (abs(pt1.y - boxes[i].max.y) < EPSILON) {
        info.normal = vec3(0.0, 1.0, 0.0);
      }
      else if (abs(pt1.y - boxes[i].min.y) < EPSILON) {
        info.normal = vec3(0.0, -1.0, 0.0);
      }
      else if (abs(pt1.z - boxes[i].max.z) < EPSILON) {
        info.normal = vec3(0.0, 0.0, 1.0);
      }
      else if (abs(pt1.z - boxes[i].min.z) < EPSILON) {
        info.normal = vec3(0.0, 0.0, -1.0);
      }
    }
  }
  return found;
}

vec2 intersectSphere(ray r, const sphere s) {
  vec3 oc = r.origin - s.center;
  float a = dot(r.dir, r.dir);
  float b = dot(oc, r.dir);
  float c = dot(oc, oc) - s.radius * s.radius;
  float h = b*b - a * c;
  return vec2((-b - sqrt(h)) / a, (-b + sqrt(h)) / a); // get intersect pts
}

bool intersectSpheres(ray r, out hitinfo info) {
  bool found = false;
  float smallest = info.lambda.x;
  for (int i = 0; i < NUM_SPHERES; i++) {
    vec2 lambda = intersectSphere(r, spheres[i]);
    if (lambda.x > 0.0 && lambda.x < lambda.y && lambda.x < smallest) {
      found = true;
      smallest = lambda.x; // sort for depth
      info.lambda = lambda;
      info.index = spheres[i].color;
      vec3 pt1 = pointAt(r, lambda.x);
      info.normal = normalize(pt1 - spheres[i].center);
    }
  }
  return found;
}

vec3 trace(ray r) {
  
  vec3 sumColor = vec3(0.0);
  vec3 kColor = vec3(1.0);
  for (int i=8; i>0; i--) {
    hitinfo info;
    
    info.lambda = vec2(MAX_SCENE_BOUNDS);
    intersectBoxes(r, info);
    intersectSpheres(r, info);
    if (info.lambda.x >= MAX_SCENE_BOUNDS) {
      break;
    }
    vec3 hit = r.origin + r.dir*info.lambda.x;

    material mat = materials[info.index];
    sumColor += kColor * mat.emissive;
    kColor *= mat.diffuse;
    vec3 lambert = info.normal + randomOnUnitSphere(hit.xy - hit.yz);
    vec3 spec = reflect(r.dir, info.normal);
    vec3 target = hit + mix(lambert, spec, mat.specular);

    if (i > 0) {
      
      r.dir = normalize(target - hit);
      r.origin = hit + r.dir * EPSILON;
    }
  }
  return sumColor;
}

void main(void) {
  ivec2 pix = ivec2(gl_GlobalInvocationID.xy);
  ivec2 size = imageSize(framebuffer);
  if (pix.x >= size.x || pix.y >= size.y) {
    return;
  }

  vec2 jitter = vec2(rand(vec2(pix.x, pix.y + count)), rand(vec2(count - pix.y, pix.x)));
  vec2 juv = (vec2(pix) + jitter) / vec2(size.x, size.y);
  vec2 uv = vec2(pix) / vec2(size.x, size.y);

  ray r;
  r.origin = camPos;
  r.dir = mix(mix(ray00.xyz, ray01.xyz, juv.y), mix(ray10.xyz, ray11.xyz, juv.y), juv.x);

  vec3 newColor = trace(r);
  newColor = clamp(vec3(0.0), vec3(1.0), pow(newColor, vec3(1.0 / 2.2)));
  vec3 color = mix(texture(tex, uv).rgb, EXPOSURE * newColor, 1.0 / (count + 1.0));

  imageStore(framebuffer, pix, vec4(color, 1.0));
}