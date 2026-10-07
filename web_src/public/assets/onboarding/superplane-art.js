/*!
 * SuperplaneArt — generative arrow engine for superplane.com
 * Usage:
 *   <script src="https://cdn.jsdelivr.net/gh/metabrand-agency/superplane@main/superplane-art.js"></script>
 *   <div id="sp-field" style="width:100%;height:520px;"></div>
 *   <script>SuperplaneArt.mount('#sp-field', {mode:'field', panel:false});</script>
 *
 * Requires three.js r128 loaded first:
 *   <script src="https://cdnjs.cloudflare.com/ajax/libs/three.js/r128/three.min.js"></script>
 *
 * mount(target, options):
 *   target  - CSS selector string or a DOM element
 *   options.mode            'field' | 'school' | 'growth' | 'network'   (default 'field')
 *   options.panel           show the utility control panel  (default false)
 *   options.allowModeSwitch show FIELD/SCHOOL/GROWTH tabs, only used if panel:true (default true)
 *   options.hiddenModes    array of mode keys to omit from the panel's tab row, e.g. ['spiral'] (only used if panel:true)
 *   options.height          CSS height if the container has none set, e.g. '520px' (default '100%')
 *   options.overrides       {field:{...}, school:{...}, growth:{...}, global:{...}} param overrides
 *   options.bgColor         override the scene/canvas background color (hex string '#RRGGBB' or number 0xRRGGBB)
 *   options.arrowColor      override the arrow base color for this instance only (hex string or number)
 *   options.arrowAccentColor override the arrow accent (near-cursor) color for this instance only; defaults to arrowColor if that is set, otherwise the shared accent color
 * Returns {destroy(){...}} handle.
 */
(function(global){
"use strict";

/* ======================================================================
   SHARED CONSTANTS
====================================================================== */
var COLOR_BASE   = 0xe7e5df;
var COLOR_ACCENT = 0xff5a1f;
var COLOR_BG     = 0x0b0b0c;
var MAX_INSTANCES = 1300;
var FONT_STACK = "'IBM Plex Mono','SFMono-Regular','Cascadia Code',Menlo,Consolas,monospace";

var MODES = {
  field: {
    label: 'FIELD PARAMS',
    params: [
      {key:'count',     label:'PARTICLES',     min:50,  max:800, step:10,  def:800},
      {key:'poles',     label:'POLE COUNT',    min:1,   max:4,   step:1,   def:2},
      {key:'strength',  label:'POLE STRENGTH', min:0.2, max:3,   step:0.05,def:1.20},
      {key:'speed',     label:'FLOW SPEED',    min:0.1, max:3,   step:0.05,def:0.3},
      {key:'cursorPull',label:'CURSOR PULL',   min:0,   max:3,   step:0.05,def:3.0}
    ]
  },
  school: {
    label: 'SCHOOL PARAMS',
    params: [
      {key:'count',      label:'AGENTS',      min:20, max:220, step:5,   def:220},
      {key:'cohesion',   label:'COHESION',    min:0,  max:2,   step:0.02,def:2.0},
      {key:'separation', label:'SEPARATION',  min:0,  max:2,   step:0.02,def:2.0},
      {key:'alignment',  label:'ALIGNMENT',   min:0,  max:2,   step:0.02,def:0.0},
      {key:'cursorPull', label:'CURSOR PULL', min:0,  max:3,   step:0.05,def:3.0},
      {key:'speed',      label:'SWIM SPEED',  min:0.1,max:3,   step:0.05,def:0.6}
    ]
  },
  growth: {
    label: 'GROWTH PARAMS',
    params: [
      {key:'maxSegments', label:'MAX SEGMENTS', min:100,max:1200,step:20, def:1200},
      {key:'maxDepth',    label:'MAX DEPTH',    min:3,  max:12,  step:1,  def:12},
      {key:'branchAngle', label:'BRANCH ANGLE', min:5,  max:60,  step:1,  def:60},
      {key:'variance',    label:'VARIANCE',     min:0,  max:1,   step:0.02,def:1.0},
      {key:'spawnChance', label:'SPAWN CHANCE', min:0,  max:1,   step:0.02,def:0.44},
      {key:'growSpeed',   label:'GROW SPEED',   min:0.2,max:4,   step:0.05,def:0.5},
      {key:'cursorPull',  label:'CURSOR BIAS',  min:0,  max:2,   step:0.05,def:0.25}
    ]
  },
  network: {
    label: 'NETWORK PARAMS',
    params: [
      {key:'count',        label:'NODES',        min:15, max:100,step:5,   def:45},
      {key:'edgesPerNode', label:'CONNECTIONS',  min:1,  max:4,  step:1,   def:2},
      {key:'curviness',    label:'CURVE AMOUNT', min:0,  max:2,  step:0.05,def:0.8},
      {key:'sway',         label:'SWAY',         min:0,  max:2,  step:0.05,def:1.0},
      {key:'swaySpeed',    label:'SWAY SPEED',   min:0.1,max:3,  step:0.05,def:0.6},
      {key:'cohesion',     label:'COHESION',     min:0,  max:2,  step:0.02,def:0.5},
      {key:'cursorPull',   label:'CURSOR PULL',  min:0,  max:3,  step:0.05,def:1.5},
      {key:'nodeSize',     label:'NODE SIZE',    min:0.1,max:2.5,step:0.05,def:0.33}
    ]
  },
  globe: {
    label: 'GLOBE PARAMS',
    params: [
      {key:'count',       label:'ARROWS',       min:100,max:1000,step:20, def:1000},
      {key:'poles',       label:'POLE COUNT',   min:2,  max:6,   step:1,  def:4},
      {key:'strength',    label:'POLE STRENGTH',min:0.2,max:3,   step:0.05,def:1.0},
      {key:'speed',       label:'FLOW SPEED',   min:0.1,max:3,   step:0.05,def:0.35},
      {key:'turbulence',  label:'TURBULENCE',   min:0,  max:2,   step:0.05,def:2.0},
      {key:'spin',        label:'PLANET SPIN',  min:0,  max:2,   step:0.05,def:0.3},
      {key:'sphereSize',  label:'SPHERE SIZE',  min:2,  max:10,  step:0.1, def:4.6},
      {key:'horizon',     label:'HORIZON',      min:0.3,max:1.0, step:0.02,def:0.64},
      {key:'offsetX',     label:'OFFSET X',     min:-6, max:6,   step:0.1, def:0},
      {key:'capAngle',    label:'VISIBLE CAP',  min:20, max:90,  step:1,  def:50},
      {key:'cursorPull',  label:'CURSOR PULL',  min:0,  max:3,   step:0.05,def:1.2}
    ]
  },
  startrek: {
    label: 'STARTREK PARAMS',
    params: [
      {key:'count',      label:'STARS',       min:200,max:1500,step:50, def:1500},
      {key:'speed',      label:'WARP SPEED',  min:0.2,max:5,   step:0.1, def:1.5},
      {key:'spread',     label:'TUNNEL WIDTH',min:1,  max:12,  step:0.2, def:5.6},
      {key:'cursorPull', label:'STEER',       min:0,  max:3,   step:0.05,def:3.0}
    ]
  },
  spiral: {
    label: 'SPIRAL PARAMS',
    params: [
      {key:'count',       label:'ARROWS',       min:100,max:1200,step:20, def:600},
      {key:'strength',    label:'ROTATION',     min:0.2,max:3,   step:0.05,def:1.2},
      {key:'turbulence',  label:'TURBULENCE',   min:0,  max:6,   step:0.1, def:1.5},
      {key:'speed',       label:'FLOW SPEED',   min:0.1,max:3,   step:0.05,def:0.6},
      {key:'sphereSize',  label:'SPHERE SIZE',  min:0.5,max:3,   step:0.05,def:1.1},
      {key:'capAngle',    label:'VISIBLE CAP',  min:25, max:75,  step:1,  def:52}
    ]
  }
};
var GLOBAL_PARAMS = [
  {key:'arrowScale',     label:'ARROW SCALE',    min:0.3,max:2.5,step:0.05,def:0.85},
  {key:'lineThickness',  label:'LINE THICKNESS', min:0.2,max:3.0,step:0.05,def:0.20},
  {key:'accentRadius',   label:'ACCENT RADIUS',  min:0,  max:8,  step:0.1, def:0.0}
];

/* ---------------- shared arrow geometry (safe to reuse across instances) ----------------
   Built lazily by ensureSharedResources() the first time mount() actually runs, so this
   file never throws or no-ops just because three.js has not executed yet at parse time
   (script load order can vary by host page).

   Two selectable arrow styles:
   - 'cone'  — a real 3D solid (cylinder shaft + cone head). Reads as an arrow from any
     camera angle, including end-on, because it's an actual 3D volume, not a flat plane.
   - 'flat'  — a thin utilitarian line-and-triangle-head shape (from the arrow2.svg
     reference), rendered as a flat 2D card. A flat card can vanish into a thin line when
     viewed edge-on, so instead of orienting it purely from the 3D direction vector, it is
     billboarded toward the camera each frame (see computeFlatOrientation): its face
     always tilts toward the camera while its length axis tracks the on-screen projection
     of the true direction — so it never degenerates into a sliver.

   Local space for both styles: length runs along local +X from -0.5 (tail) to +0.5 (tip). */
var HEAD_LEN = 0.30;      // fraction of total length occupied by the cone head
var HEAD_RADIUS = 0.16;   // local head base radius
var SHAFT_RADIUS = 0.055; // local shaft radius (this is what LINE THICKNESS scales, cone style only)
var RADIAL_SEGMENTS = 7;  // low-poly, keeps triangle count small across many instances

// 'flat' style proportions, from the arrow2.svg reference (thin hairline shaft, wide flat head)
var FLAT_SHAFT_HALF_W = 0.02;
var FLAT_HEAD_HALF_W  = 0.05;
var FLAT_HEAD_LEN     = 0.23;
var FLAT_LEG_LEN      = 0.60; // leg length is independent of head size — shortening this never grows the head

// 'chevron' style: thin leg + a head made of two line strokes (a "<" angle) at the
// SAME thickness as the leg, instead of a filled triangle.
var CHEVRON_LINE_HALF_W  = 0.02;
var CHEVRON_HEAD_LEN     = 0.22;
var CHEVRON_BARB_ANGLE   = 28; // degrees, each barb's angle off the centerline

var ARROW_STYLES = [
  {key:'cone',    label:'CONE (3D)'},
  {key:'flat',    label:'FLAT (UTILITARIAN)'},
  {key:'chevron', label:'CHEVRON (LINES)'}
];
// styles that are flat 2D cards needing camera-facing billboarding, and where only
// ARROW SCALE applies (no separate LINE THICKNESS control) — as opposed to 'cone',
// which is a real 3D volume with its own thickness.
var FLAT_STYLE_KEYS = {flat:true, chevron:true};

// Each mode resets to its own known-good composition when selected from the tabs —
// arrow style, FLAT, ARROW SCALE and the camera angle. GLOBE and STARTREK depend on
// real depth and on the CONE style specifically (a flat/billboarded arrow can't sit
// flush on a curved globe surface, and only a real 3D cone reads as a point when it
// points straight at the camera in STARTREK) — so for those two, ARROW STYLE and
// FLAT are also locked (disabled in the UI) rather than just defaulted.
var MODE_SCENE_DEFAULTS = {
  field:    {arrowStyle:'flat', flatMode:true,  arrowScale:0.85, cam:{theta:356.6,phi:92.6,radius:6.62}, restrictStyle:false},
  school:   {arrowStyle:'flat', flatMode:true,  arrowScale:0.85, cam:{theta:356.6,phi:92.6,radius:6.62}, restrictStyle:false},
  growth:   {arrowStyle:'flat', flatMode:true,  arrowScale:0.85, cam:{theta:356.6,phi:92.6,radius:6.62}, restrictStyle:false},
  network:  {arrowStyle:'flat', flatMode:true,  arrowScale:0.85, cam:{theta:356.6,phi:92.6,radius:6.62}, restrictStyle:false},
  globe:    {arrowStyle:'cone', flatMode:false, arrowScale:0.50, cam:{theta:82.9, phi:72.0, radius:6.62}, restrictStyle:true},
  spiral:   {arrowStyle:'cone', flatMode:false, arrowScale:0.26, cam:{theta:0,    phi:90,   radius:5.2},  restrictStyle:true},
  startrek: {arrowStyle:'cone', flatMode:false, arrowScale:0.80, cam:{theta:0, phi:90, radius:6.62}, restrictStyle:true}
};

var ARROW_LOCAL_POINTS_BY_STYLE = {}; // style key -> deduplicated local vertices, used for SVG silhouette export
var CONE_SILHOUETTE_POINTS = null; // ordered concave outline for the cone style's true silhouette (see buildConeArrowGeometry)
var sharedArrowGeoByStyle = {};       // style key -> BufferGeometry
var sharedBaseMat = null, sharedAccentMat = null, sharedPoleGeo = null, sharedPoleMat = null;
var sharedNodeGeo = null, sharedLineMatBase = null, sharedLineMatAccent = null;
var MAX_NETWORK_NODES = 150, MAX_NETWORK_EDGES = 400, EDGE_SEGMENTS = 14;

function mergeGeometries(geoList){
  var totalVerts = 0;
  var nonIndexed = geoList.map(function(g){ return g.index ? g.toNonIndexed() : g; });
  nonIndexed.forEach(function(g){ totalVerts += g.attributes.position.count; });
  var positions = new Float32Array(totalVerts*3);
  var normals = new Float32Array(totalVerts*3);
  var offset = 0;
  nonIndexed.forEach(function(g){
    var p = g.attributes.position.array;
    positions.set(p, offset*3);
    if(g.attributes.normal){ normals.set(g.attributes.normal.array, offset*3); }
    offset += g.attributes.position.count;
  });
  var merged = new THREE.BufferGeometry();
  merged.setAttribute('position', new THREE.Float32BufferAttribute(positions, 3));
  merged.setAttribute('normal', new THREE.Float32BufferAttribute(normals, 3));
  return merged;
}

function dedupPoints(float32arr){
  var seen = {}, out = [];
  for(var i=0;i<float32arr.length;i+=3){
    var x=float32arr[i], y=float32arr[i+1], z=float32arr[i+2];
    var key = x.toFixed(4)+','+y.toFixed(4)+','+z.toFixed(4);
    if(!seen[key]){ seen[key]=true; out.push(new THREE.Vector3(x,y,z)); }
  }
  return out;
}

function buildConeArrowGeometry(){
  var shaftLen = 1 - HEAD_LEN;
  var shaftCenterX = -HEAD_LEN/2;
  var headCenterX = 0.5 - HEAD_LEN/2;

  var shaftGeo = new THREE.CylinderGeometry(SHAFT_RADIUS, SHAFT_RADIUS, shaftLen, RADIAL_SEGMENTS);
  shaftGeo.rotateZ(-Math.PI/2); // align cylinder's height axis (Y) to local +X
  shaftGeo.translate(shaftCenterX, 0, 0);

  var headGeo = new THREE.ConeGeometry(HEAD_RADIUS, HEAD_LEN, RADIAL_SEGMENTS);
  headGeo.rotateZ(-Math.PI/2); // cone apex (was +Y) now points toward +X (the tip)
  headGeo.translate(headCenterX, 0, 0);

  var merged = mergeGeometries([shaftGeo, headGeo]);
  ARROW_LOCAL_POINTS_BY_STYLE.cone = dedupPoints(merged.attributes.position.array);

  // A cylinder+cone is a solid of revolution, so -- viewed from any angle that
  // isn't dead-on down its own axis -- its silhouette is always this exact
  // profile curve mirrored top/bottom, regardless of roll around the axis.
  // Head wider than shaft means the shoulder is a reflex (concave) corner,
  // same as the 'flat' style: a convex hull of the mesh's vertices fills that
  // notch in and turns the silhouette into a blob. Store this ordered outline
  // for SVG export to use directly (billboarded, no hull), exactly like 'flat'.
  var headStart = 0.5 - HEAD_LEN;
  CONE_SILHOUETTE_POINTS = [
    new THREE.Vector3(-0.5,      SHAFT_RADIUS, 0), // tail-top
    new THREE.Vector3(headStart, SHAFT_RADIUS, 0), // shoulder-top-inner
    new THREE.Vector3(headStart, HEAD_RADIUS,  0), // shoulder-top-outer
    new THREE.Vector3( 0.5,      0,            0), // tip
    new THREE.Vector3(headStart,-HEAD_RADIUS,  0), // shoulder-bottom-outer
    new THREE.Vector3(headStart,-SHAFT_RADIUS, 0), // shoulder-bottom-inner
    new THREE.Vector3(-0.5,     -SHAFT_RADIUS, 0)  // tail-bottom
  ];

  return merged;
}

function buildFlatArrowGeometry(){
  var sw = FLAT_SHAFT_HALF_W, hw = FLAT_HEAD_HALF_W, headStart = 0.5 - FLAT_HEAD_LEN;
  var tailX = headStart - FLAT_LEG_LEN; // leg length is independent of head size
  var outline = [
    new THREE.Vector3(tailX,      sw, 0), // 0 tail-top
    new THREE.Vector3(headStart, sw, 0), // 1 shoulder-top-inner
    new THREE.Vector3(headStart, hw, 0), // 2 shoulder-top-outer
    new THREE.Vector3( 0.5,      0,  0), // 3 tip
    new THREE.Vector3(headStart,-hw, 0), // 4 shoulder-bottom-outer
    new THREE.Vector3(headStart,-sw, 0), // 5 shoulder-bottom-inner
    new THREE.Vector3(tailX,     -sw, 0)  // 6 tail-bottom
  ];
  // Correct decomposition of this concave (notched) polygon: the shaft is a plain
  // rectangle (tail corners + inner shoulder corners) and the head is one triangle
  // (outer shoulder corners + tip). A naive single-vertex fan across all 7 points
  // is WRONG here — vertices 1 and 5 are reflex (concave) corners, so fan triangles
  // through them bulge outside the true silhouette (this is what produced the
  // two-triangle "not an arrow" shape). Winding is reversed (…,1,0 / …,5,1 / …,4,3)
  // so the face normal points to local +Z, which is the side the billboard faces
  // toward the camera.
  var tris = [[6,1,0],[6,5,1],[2,4,3]];
  var positions = [];
  tris.forEach(function(t){
    t.forEach(function(i){ var p=outline[i]; positions.push(p.x,p.y,p.z); });
  });
  var geo = new THREE.BufferGeometry();
  geo.setAttribute('position', new THREE.Float32BufferAttribute(positions, 3));
  geo.computeVertexNormals();
  ARROW_LOCAL_POINTS_BY_STYLE.flat = outline;
  return geo;
}

function quadAlongSegment2D(ax, ay, bx, by, halfWidth){
  var dx=bx-ax, dy=by-ay;
  var len = Math.sqrt(dx*dx+dy*dy) || 1;
  var nx = -dy/len*halfWidth, ny = dx/len*halfWidth;
  return [
    new THREE.Vector3(ax+nx, ay+ny, 0),
    new THREE.Vector3(bx+nx, by+ny, 0),
    new THREE.Vector3(bx-nx, by-ny, 0),
    new THREE.Vector3(ax-nx, ay-ny, 0)
  ];
}
function pushQuadTris(positions, q){
  // winding order (0,2,1)+(0,3,2) gives a +Z-facing normal for this quad layout
  [q[0],q[2],q[1], q[0],q[3],q[2]].forEach(function(p){ positions.push(p.x,p.y,p.z); });
}

function buildChevronArrowGeometry(){
  var w = CHEVRON_LINE_HALF_W;
  var headStart = 0.5 - CHEVRON_HEAD_LEN;
  var spread = CHEVRON_HEAD_LEN * Math.tan(CHEVRON_BARB_ANGLE*Math.PI/180);

  // Run the shaft's centerline all the way up to the tip, so its top edge lines up
  // with the chevron's own peak instead of stopping short and leaving open space
  // between the leg and the "V".
  var shaftQuad   = quadAlongSegment2D(-0.5, 0, 0.5, 0, w);
  var barbTopQuad = quadAlongSegment2D(0.5, 0, headStart,  spread, w);
  var barbBotQuad = quadAlongSegment2D(0.5, 0, headStart, -spread, w);

  var positions = [];
  pushQuadTris(positions, shaftQuad);
  pushQuadTris(positions, barbTopQuad);
  pushQuadTris(positions, barbBotQuad);

  var geo = new THREE.BufferGeometry();
  geo.setAttribute('position', new THREE.Float32BufferAttribute(positions, 3));
  geo.computeVertexNormals();
  ARROW_LOCAL_POINTS_BY_STYLE.chevron = shaftQuad.concat(barbTopQuad, barbBotQuad);
  return geo;
}

function ensureSharedResources(){
  if(sharedBaseMat) return; // already built
  sharedArrowGeoByStyle.cone = buildConeArrowGeometry();
  sharedArrowGeoByStyle.flat = buildFlatArrowGeometry();
  sharedArrowGeoByStyle.chevron = buildChevronArrowGeometry();
  sharedBaseMat = new THREE.MeshBasicMaterial({color:COLOR_BASE});
  sharedAccentMat = new THREE.MeshBasicMaterial({color:COLOR_ACCENT});
  sharedPoleGeo = new THREE.RingGeometry(0.04,0.06,16);
  sharedPoleMat = new THREE.MeshBasicMaterial({color:0x5c5c58, side:THREE.DoubleSide});
  sharedNodeGeo = new THREE.PlaneGeometry(1,1);
  sharedLineMatBase = new THREE.LineBasicMaterial({color:COLOR_BASE, transparent:true, opacity:0.55});
  sharedLineMatAccent = new THREE.LineBasicMaterial({color:COLOR_ACCENT, transparent:true, opacity:0.85});
}

/* ---------------- standard camera-facing sprite billboard (for NETWORK node rectangles) ----------------
   Unlike computeFlatOrientation (which tracks a direction vector), a node rectangle has
   no direction — it just faces the camera, upright, like a classic billboarded sprite. */
var SB_camDir=null, SB_x=null, SB_y=null, SB_up=null, SB_mat=null;
function computeSpriteOrientation(pos, cameraPosition, outQuat){
  if(!SB_camDir){
    SB_camDir=new THREE.Vector3(); SB_x=new THREE.Vector3(); SB_y=new THREE.Vector3();
    SB_up=new THREE.Vector3(0,1,0); SB_mat=new THREE.Matrix4();
  }
  SB_camDir.subVectors(cameraPosition, pos);
  if(SB_camDir.lengthSq()<1e-8) SB_camDir.set(0,0,1);
  SB_camDir.normalize();
  SB_x.crossVectors(SB_up, SB_camDir);
  if(SB_x.lengthSq()<1e-6) SB_x.set(1,0,0); else SB_x.normalize();
  SB_y.crossVectors(SB_camDir, SB_x).normalize();
  SB_mat.makeBasis(SB_x, SB_y, SB_camDir);
  outQuat.setFromRotationMatrix(SB_mat);
}

/* ---------------- camera-facing orientation for the 'flat' style ----------------
   A flat 2D card can vanish into a sliver when viewed edge-on, so instead of
   orienting it purely from the 3D direction vector, tilt its face toward the camera
   each frame and only use the direction vector to choose which way it points *within*
   that camera-facing plane (i.e. the on-screen projection of the true direction). */
var FA_camDir = null, FA_x = null, FA_y = null, FA_up = null, FA_mat = null;
function computeFlatOrientation(pos, dir, cameraPosition, outQuat){
  if(!FA_camDir){
    FA_camDir = new THREE.Vector3(); FA_x = new THREE.Vector3();
    FA_y = new THREE.Vector3(); FA_up = new THREE.Vector3(0,1,0);
    FA_mat = new THREE.Matrix4();
  }
  FA_camDir.subVectors(cameraPosition, pos);
  if(FA_camDir.lengthSq() < 1e-8) FA_camDir.set(0,0,1);
  FA_camDir.normalize();

  FA_x.copy(dir).addScaledVector(FA_camDir, -dir.dot(FA_camDir));
  if(FA_x.lengthSq() < 1e-6){
    FA_x.copy(FA_up).addScaledVector(FA_camDir, -FA_up.dot(FA_camDir));
    if(FA_x.lengthSq() < 1e-6) FA_x.set(1,0,0);
  }
  FA_x.normalize();

  FA_y.crossVectors(FA_camDir, FA_x).normalize();
  FA_x.crossVectors(FA_y, FA_camDir).normalize(); // re-orthogonalize

  FA_mat.makeBasis(FA_x, FA_y, FA_camDir);
  outQuat.setFromRotationMatrix(FA_mat);
}

/* ---------------- 2D convex hull (Andrew's monotone chain) — used to silhouette the
   3D arrow solid into a flat SVG polygon on export ---------------- */
function convexHull2D(pts){
  pts = pts.slice().sort(function(a,b){ return a[0]-b[0] || a[1]-b[1]; });
  var n = pts.length;
  if(n < 3) return pts;
  function cross(o,a,b){ return (a[0]-o[0])*(b[1]-o[1]) - (a[1]-o[1])*(b[0]-o[0]); }
  var lower = [];
  for(var i=0;i<n;i++){
    while(lower.length>=2 && cross(lower[lower.length-2], lower[lower.length-1], pts[i])<=0) lower.pop();
    lower.push(pts[i]);
  }
  var upper = [];
  for(var i=n-1;i>=0;i--){
    while(upper.length>=2 && cross(upper[upper.length-2], upper[upper.length-1], pts[i])<=0) upper.pop();
    upper.push(pts[i]);
  }
  upper.pop(); lower.pop();
  return lower.concat(upper);
}

/* ---------------- shared slider row builder ---------------- */
function makeRow(param, getVal, setVal, onChange){
  var row = document.createElement('div');
  row.style.cssText = 'display:flex;align-items:center;gap:8px;margin-bottom:9px;font-family:'+FONT_STACK+';font-size:11px;';
  var val = document.createElement('span');
  val.style.cssText = 'width:38px;flex:0 0 38px;text-align:left;font-variant-numeric:tabular-nums;color:#141414;';
  var input = document.createElement('input');
  input.type='range'; input.min=param.min; input.max=param.max; input.step=param.step; input.value=getVal();
  input.style.cssText = '-webkit-appearance:none;appearance:none;flex:1 1 auto;height:14px;border:1px solid #1c1c1c;background:#cfcfcb;cursor:pointer;';
  var label = document.createElement('span');
  label.style.cssText = 'flex:0 0 auto;white-space:nowrap;letter-spacing:0.3px;color:#3a3a38;';
  label.textContent = param.label;

  function refresh(){
    var v = parseFloat(input.value);
    var isInt = Math.abs(param.step - Math.round(param.step)) < 1e-9 && param.step >= 1;
    val.textContent = isInt ? String(Math.round(v)) : v.toFixed(2);
    var pct = (v-param.min)/(param.max-param.min)*100;
    input.style.background = 'linear-gradient(to right, #141414 0%, #141414 '+pct+'%, #cfcfcb '+pct+'%, #cfcfcb 100%)';
  }
  input.addEventListener('input', function(){
    setVal(parseFloat(input.value));
    refresh();
    if(onChange) onChange();
  });
  refresh();
  row.appendChild(val); row.appendChild(input); row.appendChild(label);
  row.rangeInput = input;
  input.setExternalValue = function(v){ input.value = v; setVal(v); refresh(); };
  return row;
}

function injectSliderThumbCSS(){
  if(document.getElementById('sp-art-thumb-css')) return;
  var style = document.createElement('style');
  style.id = 'sp-art-thumb-css';
  style.textContent =
    '.sp-art-root input[type=range]::-webkit-slider-thumb{-webkit-appearance:none;width:2px;height:14px;background:transparent;border:none;}'+
    '.sp-art-root input[type=range]::-moz-range-thumb{width:2px;height:14px;background:transparent;border:none;}'+
    '.sp-art-root input[type=range]::-moz-range-track{height:14px;background:transparent;}';
  document.head.appendChild(style);
}

/* ---------------- shared code-snippet modal (used by the EXPORT EMBED button) ---------------- */
function showCodeModal(title, code){
  var overlay = document.createElement('div');
  overlay.style.cssText = 'position:fixed;inset:0;background:rgba(11,11,12,0.6);z-index:2147483647;display:flex;align-items:center;justify-content:center;font-family:'+FONT_STACK+';';
  var box = document.createElement('div');
  box.style.cssText = 'background:#ececea;border:1px solid #1c1c1c;width:min(640px,92vw);max-height:82vh;display:flex;flex-direction:column;padding:16px;box-sizing:border-box;';
  var titleEl = document.createElement('div');
  titleEl.textContent = title;
  titleEl.style.cssText = 'font-weight:700;letter-spacing:0.5px;margin-bottom:10px;font-size:11px;color:#141414;';
  var ta = document.createElement('textarea');
  ta.value = code;
  ta.readOnly = true;
  ta.spellcheck = false;
  ta.style.cssText = 'flex:1;min-height:220px;width:100%;box-sizing:border-box;font-family:'+FONT_STACK+';font-size:11px;line-height:1.5;padding:10px;border:1px solid #1c1c1c;resize:vertical;background:#fff;color:#141414;';
  var btnRow = document.createElement('div');
  btnRow.style.cssText = 'display:flex;gap:8px;margin-top:10px;';
  var copyBtn = document.createElement('button');
  copyBtn.textContent = 'COPY';
  var closeBtn = document.createElement('button');
  closeBtn.textContent = 'CLOSE';
  [copyBtn, closeBtn].forEach(function(b){
    b.style.cssText = 'flex:1;padding:8px 4px;border:1px solid #1c1c1c;background:#ececea;color:#141414;font-family:inherit;font-size:10px;letter-spacing:0.4px;cursor:pointer;';
  });
  copyBtn.addEventListener('click', function(){
    ta.focus(); ta.select();
    var ok = false;
    try{
      if(navigator.clipboard && navigator.clipboard.writeText){
        navigator.clipboard.writeText(code);
        ok = true;
      }
    }catch(e){}
    if(!ok){ try{ document.execCommand('copy'); }catch(e){} }
    copyBtn.textContent = 'COPIED';
    setTimeout(function(){ copyBtn.textContent = 'COPY'; }, 1500);
  });
  function close(){ if(overlay.parentNode) overlay.parentNode.removeChild(overlay); }
  closeBtn.addEventListener('click', close);
  overlay.addEventListener('click', function(e){ if(e.target===overlay) close(); });
  btnRow.appendChild(copyBtn); btnRow.appendChild(closeBtn);
  box.appendChild(titleEl); box.appendChild(ta); box.appendChild(btnRow);
  overlay.appendChild(box);
  document.body.appendChild(overlay);
  ta.focus(); ta.select();
}

/* ======================================================================
   MOUNT
====================================================================== */
function mount(target, options){
  options = options || {};
  var container = (typeof target === 'string') ? document.querySelector(target) : target;
  if(!container){ console.error('SuperplaneArt: target not found:', target); return null; }

  if(typeof THREE === 'undefined'){
    // three.js has not executed yet (script load order can vary by host page) — retry briefly
    // instead of failing silently, so a temporary race never leaves a blank container.
    var attempts = 0;
    var waitId = setInterval(function(){
      attempts++;
      if(typeof THREE !== 'undefined'){
        clearInterval(waitId);
        mount(target, options);
      } else if(attempts > 50){ // ~5s
        clearInterval(waitId);
        console.error('SuperplaneArt: three.js did not load within 5s.');
        container.innerHTML = '<div style="padding:16px;font-family:monospace;font-size:12px;color:#a33;">SuperplaneArt: three.js failed to load.</div>';
      }
    }, 100);
    return null;
  }
  ensureSharedResources();

  // options.bgColor / options.arrowColor / options.arrowAccentColor let a single
  // mount() instance override the module-wide defaults (COLOR_BG / COLOR_BASE /
  // COLOR_ACCENT) without affecting any other SuperplaneArt instance on the same
  // page — the shared base/accent materials built by ensureSharedResources() stay
  // untouched and are still reused whenever a mount doesn't ask for a custom color.
  // Accepts either a hex string ('#EF8D0B') or a hex number (0xEF8D0B) — anything
  // THREE.Color itself accepts.
  var effectiveBgColor = (typeof options.bgColor !== 'undefined') ? options.bgColor : COLOR_BG;
  var instanceBaseMat = (typeof options.arrowColor !== 'undefined')
    ? new THREE.MeshBasicMaterial({color: options.arrowColor})
    : sharedBaseMat;
  var instanceAccentMat = (typeof options.arrowAccentColor !== 'undefined')
    ? new THREE.MeshBasicMaterial({color: options.arrowAccentColor})
    : (typeof options.arrowColor !== 'undefined' ? instanceBaseMat : sharedAccentMat);

  injectSliderThumbCSS();
  container.classList.add('sp-art-root');
  container.innerHTML = '';
  if(!container.style.height) container.style.height = options.height || '100%';
  if(!container.style.width) container.style.width = '100%';
  container.style.position = container.style.position || 'relative';
  container.style.overflow = 'hidden';

  var showPanel = !!options.panel;
  var allowModeSwitch = options.allowModeSwitch !== false;

  var cfg = {global:{}, field:{}, school:{}, growth:{}, network:{}, globe:{}, startrek:{}, spiral:{}};
  GLOBAL_PARAMS.forEach(function(p){ cfg.global[p.key] = p.def; });
  cfg.global.arrowStyle = 'flat';
  Object.keys(MODES).forEach(function(m){
    MODES[m].params.forEach(function(p){ cfg[m][p.key] = p.def; });
  });

  var state = { mode: options.mode || 'field', autoRotate: false };

  // Apply this mode's own scene defaults (arrow style, FLAT, ARROW SCALE, camera) as
  // the baseline BEFORE any explicit options/overrides are layered on — this is what
  // keeps GLOBE/STARTREK/SPIRAL correct even when mounted directly into that mode
  // (e.g. a standalone embed), not only when switched into via the panel tabs.
  var initialSceneDefaults = MODE_SCENE_DEFAULTS[state.mode];
  var initialFlatModeDefault = true;
  var initialCamDefault = null;
  if(initialSceneDefaults){
    cfg.global.arrowStyle = initialSceneDefaults.arrowStyle;
    cfg.global.arrowScale = initialSceneDefaults.arrowScale;
    initialFlatModeDefault = initialSceneDefaults.flatMode;
    initialCamDefault = initialSceneDefaults.cam;
  }

  if(options.overrides){
    Object.keys(options.overrides).forEach(function(section){
      if(cfg[section]) Object.assign(cfg[section], options.overrides[section]);
    });
  }

  /* ---------------- DOM ---------------- */
  var root = document.createElement('div');
  root.style.cssText = 'display:flex;width:100%;height:100%;background:#0b0b0c;font-family:'+FONT_STACK+';font-size:11px;color:#141414;-webkit-font-smoothing:none;';
  container.appendChild(root);

  var panelEl = null, modeParamsEl = null, modeSectionLabel = null, readoutEl = null, tabsEls = {};
  var camCoordsEl = null;
  var cameraLocked = options.cameraLocked !== false;
  var interactiveEnabled = options.interactive !== false;
  var flatMode = (typeof options.flatMode === 'boolean') ? options.flatMode : initialFlatModeDefault;
  var FLAT_Z_SQUASH = 0.04; // how much world-space depth remains when FLAT is checked
  var renderPosScratch = new THREE.Vector3();
  if(showPanel){
    panelEl = document.createElement('div');
    panelEl.style.cssText = 'width:280px;min-width:280px;height:100%;overflow-y:auto;background:#ececea;border-right:1px solid #1c1c1c;padding:14px;box-sizing:border-box;';
    root.appendChild(panelEl);

    var logoImg = document.createElement('img');
    logoImg.src = 'https://cdn.prod.website-files.com/6aa50db8e89996c95ee03309/6aa9537dce3c2279451bebed_superplane.svg';
    logoImg.alt = 'SuperPlane';
    logoImg.style.cssText = 'display:block;height:20px;width:auto;margin-bottom:60px;opacity:0.85;';
    panelEl.appendChild(logoImg);

    if(allowModeSwitch){
      var tabsRow = document.createElement('div');
      tabsRow.style.cssText = 'display:grid;grid-template-columns:repeat(3, 1fr);border:1px solid #1c1c1c;border-right:none;border-bottom:none;margin-bottom:14px;';
      var MODE_KEYS = ['field','school','growth','network','globe','startrek','spiral'];
      if(options.hiddenModes && options.hiddenModes.length){
        MODE_KEYS = MODE_KEYS.filter(function(m){ return options.hiddenModes.indexOf(m) === -1; });
      }
      MODE_KEYS.forEach(function(m, i){
        var t = document.createElement('div');
        t.textContent = m.toUpperCase();
        t.style.cssText = 'text-align:center;padding:7px 0;cursor:pointer;user-select:none;'+
          'border-right:1px solid #1c1c1c;border-bottom:1px solid #1c1c1c;'+
          'background:'+(m===state.mode?'#141414':'#ececea')+';color:'+(m===state.mode?'#ececea':'#141414')+';';
        t.addEventListener('click', function(){
          state.mode = m;
          Object.keys(tabsEls).forEach(function(k){
            tabsEls[k].style.background = (k===m)?'#141414':'#ececea';
            tabsEls[k].style.color = (k===m)?'#ececea':'#141414';
          });
          rebuildModeParams();
          updateArrowGlobalSectionVisibility();
          // Apply the new camera angle FIRST and update camera.position immediately
          // (not on the next animation frame) — STARTREK derives its "forward" flight
          // axis from camera.position at reset time, so resetting the sim before the
          // camera has actually moved would fly everyone off in the old direction.
          applyModeSceneDefaults();
          applyOrbit();
          resetSimForMode();
        });
        tabsEls[m] = t;
        tabsRow.appendChild(t);
      });
      panelEl.appendChild(tabsRow);
    }

    var globalLabel = document.createElement('div');
    globalLabel.textContent = 'GLOBAL';
    globalLabel.style.cssText = 'font-weight:700;letter-spacing:0.5px;margin-bottom:8px;';
    panelEl.appendChild(globalLabel);

    var styleRow = document.createElement('div');
    styleRow.style.cssText = 'display:flex;align-items:center;gap:8px;margin-bottom:9px;';
    var styleLabel = document.createElement('span');
    styleLabel.textContent = 'ARROW STYLE';
    styleLabel.style.cssText = 'flex:0 0 auto;color:#3a3a38;letter-spacing:0.3px;';
    var styleSelect = document.createElement('select');
    styleSelect.style.cssText = 'flex:1 1 auto;height:24px;border:1px solid #1c1c1c;background:#fff;'+
      'color:#141414;font-family:'+FONT_STACK+';font-size:11px;min-width:0;';
    ARROW_STYLES.forEach(function(s){
      var opt = document.createElement('option');
      opt.value = s.key; opt.textContent = s.label;
      styleSelect.appendChild(opt);
    });
    styleSelect.value = cfg.global.arrowStyle;
    styleSelect.addEventListener('change', function(){
      cfg.global.arrowStyle = styleSelect.value;
      updateLineThicknessVisibility();
    });
    styleRow.appendChild(styleLabel); styleRow.appendChild(styleSelect);
    panelEl.appendChild(styleRow);

    var globalParamsEl = document.createElement('div');
    panelEl.appendChild(globalParamsEl);
    var lineThicknessRowEl = null, arrowScaleRowEl = null;
    GLOBAL_PARAMS.forEach(function(p){
      var row = makeRow(p, function(){return cfg.global[p.key];}, function(v){cfg.global[p.key]=v;});
      globalParamsEl.appendChild(row);
      if(p.key === 'lineThickness') lineThicknessRowEl = row;
      if(p.key === 'arrowScale') arrowScaleRowEl = row;
    });
    function updateLineThicknessVisibility(){
      if(lineThicknessRowEl) lineThicknessRowEl.style.display = FLAT_STYLE_KEYS[cfg.global.arrowStyle] ? 'none' : 'flex';
    }
    updateLineThicknessVisibility();
    function updateArrowGlobalSectionVisibility(){
      // ARROW STYLE / ARROW SCALE are meaningless for NETWORK (rectangles + curves,
      // sized by its own NODE SIZE param instead) — hide them in that mode.
      var isNetwork = state.mode === 'network';
      styleRow.style.display = isNetwork ? 'none' : 'flex';
      if(arrowScaleRowEl) arrowScaleRowEl.style.display = isNetwork ? 'none' : 'flex';
      if(!isNetwork) updateLineThicknessVisibility();
      else if(lineThicknessRowEl) lineThicknessRowEl.style.display = 'none';
    }
    updateArrowGlobalSectionVisibility();

    // Reset to this mode's own known-good composition every time it's selected —
    // style, FLAT, ARROW SCALE and camera angle — rather than carrying over whatever
    // was left from the previous tab. GLOBE/STARTREK also lock ARROW STYLE and FLAT
    // in the UI (disabled, not just defaulted) since those two only work with a real
    // 3D CONE and full depth.
    function applyStyleRestrictionUI(){
      var d = MODE_SCENE_DEFAULTS[state.mode];
      var restrict = d ? d.restrictStyle : false;
      styleSelect.value = cfg.global.arrowStyle;
      styleSelect.disabled = restrict;
      styleSelect.style.opacity = restrict ? '0.5' : '1';
      styleSelect.style.cursor = restrict ? 'not-allowed' : 'pointer';
      flatCheckbox.checked = flatMode;
      flatCheckbox.disabled = restrict;
      flatRow.style.opacity = restrict ? '0.5' : '1';
      flatRow.style.cursor = restrict ? 'not-allowed' : 'pointer';
    }
    function applyModeSceneDefaults(){
      var d = MODE_SCENE_DEFAULTS[state.mode];
      if(!d) return;
      cfg.global.arrowStyle = d.arrowStyle;
      flatMode = d.flatMode;
      if(arrowScaleRowEl && arrowScaleRowEl.rangeInput) arrowScaleRowEl.rangeInput.setExternalValue(d.arrowScale);
      else cfg.global.arrowScale = d.arrowScale;
      orbit.theta = d.cam.theta*Math.PI/180;
      orbit.phi = d.cam.phi*Math.PI/180;
      orbit.radius = d.cam.radius;
      applyStyleRestrictionUI();
      updateLineThicknessVisibility();
    }

    var btnRow = document.createElement('div');
    btnRow.style.cssText = 'display:flex;gap:8px;margin:8px 0 14px;';
    var btnRotate = document.createElement('button');
    btnRotate.textContent = 'AUTO ROTATE';
    var btnReset = document.createElement('button');
    btnReset.textContent = 'RESET';
    [btnRotate, btnReset].forEach(function(b){
      b.style.cssText = 'flex:1;padding:8px 4px;border:1px solid #1c1c1c;background:#ececea;color:#141414;font-family:inherit;font-size:10px;letter-spacing:0.4px;cursor:pointer;';
    });
    btnRotate.addEventListener('click', function(){
      state.autoRotate = !state.autoRotate;
      btnRotate.style.background = state.autoRotate ? '#ff5a1f' : '#ececea';
      btnRotate.style.borderColor = state.autoRotate ? '#ff5a1f' : '#1c1c1c';
    });
    btnReset.addEventListener('click', function(){
      var d = MODE_SCENE_DEFAULTS[state.mode];
      if(d){ orbit.theta = d.cam.theta*Math.PI/180; orbit.phi = d.cam.phi*Math.PI/180; orbit.radius = d.cam.radius; }
      applyOrbit();
      resetSimForMode();
    });
    btnRow.appendChild(btnRotate); btnRow.appendChild(btnReset);
    panelEl.appendChild(btnRow);

    var lockRow = document.createElement('label');
    lockRow.style.cssText = 'display:flex;align-items:center;gap:8px;margin-bottom:8px;cursor:pointer;user-select:none;color:#141414;';
    var lockCheckbox = document.createElement('input');
    lockCheckbox.type = 'checkbox';
    lockCheckbox.style.cssText = 'width:14px;height:14px;accent-color:#141414;cursor:pointer;flex:0 0 auto;';
    var lockLabelText = document.createElement('span');
    lockLabelText.textContent = 'LOCK CAMERA ROTATION';
    lockCheckbox.checked = cameraLocked;
    lockCheckbox.addEventListener('change', function(){ cameraLocked = lockCheckbox.checked; });
    lockRow.appendChild(lockCheckbox); lockRow.appendChild(lockLabelText);
    panelEl.appendChild(lockRow);

    var flatRow = document.createElement('label');
    flatRow.style.cssText = 'display:flex;align-items:center;gap:8px;margin-bottom:8px;cursor:pointer;user-select:none;color:#141414;';
    var flatCheckbox = document.createElement('input');
    flatCheckbox.type = 'checkbox';
    flatCheckbox.style.cssText = 'width:14px;height:14px;accent-color:#141414;cursor:pointer;flex:0 0 auto;';
    var flatLabelText = document.createElement('span');
    flatLabelText.textContent = 'FLAT (SQUASH DEPTH)';
    flatCheckbox.checked = flatMode;
    flatCheckbox.addEventListener('change', function(){ flatMode = flatCheckbox.checked; });
    flatRow.appendChild(flatCheckbox); flatRow.appendChild(flatLabelText);
    panelEl.appendChild(flatRow);

    applyStyleRestrictionUI();

    camCoordsEl = document.createElement('div');
    camCoordsEl.style.cssText = 'margin-bottom:14px;line-height:1.6;color:#3a3a38;';
    panelEl.appendChild(camCoordsEl);

    var divider1 = document.createElement('div');
    divider1.style.cssText = 'border-top:1px solid #1c1c1c;margin:14px 0;';
    panelEl.appendChild(divider1);

    modeSectionLabel = document.createElement('div');
    modeSectionLabel.style.cssText = 'font-weight:700;letter-spacing:0.5px;margin-bottom:8px;';
    panelEl.appendChild(modeSectionLabel);
    modeParamsEl = document.createElement('div');
    panelEl.appendChild(modeParamsEl);

    var divider1b = document.createElement('div');
    divider1b.style.cssText = 'border-top:1px solid #1c1c1c;margin:14px 0;';
    panelEl.appendChild(divider1b);

    var colorsLabel = document.createElement('div');
    colorsLabel.textContent = 'COLORS';
    colorsLabel.style.cssText = 'font-weight:700;letter-spacing:0.5px;margin-bottom:8px;';
    panelEl.appendChild(colorsLabel);

    function toHexString(c){
      return '#' + new THREE.Color(c).getHexString();
    }
    var colorsRow = document.createElement('div');
    colorsRow.style.cssText = 'display:flex;gap:8px;margin-bottom:14px;align-items:center;';
    var arrowColorLabel = document.createElement('label');
    arrowColorLabel.textContent = 'ARROW';
    arrowColorLabel.style.cssText = 'font-size:9px;letter-spacing:0.4px;color:#9a9a94;display:block;margin-bottom:3px;';
    var arrowColorInput = document.createElement('input');
    arrowColorInput.type = 'text';
    arrowColorInput.maxLength = 7;
    arrowColorInput.value = toHexString(instanceBaseMat.color);
    arrowColorInput.style.cssText = 'width:100%;padding:6px 4px;border:1px solid #1c1c1c;background:#141414;color:#ececea;font-family:inherit;font-size:10px;box-sizing:border-box;';
    var arrowColorWrap = document.createElement('div');
    arrowColorWrap.style.cssText = 'flex:1;';
    arrowColorWrap.appendChild(arrowColorLabel); arrowColorWrap.appendChild(arrowColorInput);

    var bgColorLabel = document.createElement('label');
    bgColorLabel.textContent = 'BACKGROUND';
    bgColorLabel.style.cssText = 'font-size:9px;letter-spacing:0.4px;color:#9a9a94;display:block;margin-bottom:3px;';
    var bgColorInput = document.createElement('input');
    bgColorInput.type = 'text';
    bgColorInput.maxLength = 7;
    bgColorInput.value = toHexString(effectiveBgColor);
    bgColorInput.style.cssText = 'width:100%;padding:6px 4px;border:1px solid #1c1c1c;background:#141414;color:#ececea;font-family:inherit;font-size:10px;box-sizing:border-box;';
    var bgColorWrap = document.createElement('div');
    bgColorWrap.style.cssText = 'flex:1;';
    bgColorWrap.appendChild(bgColorLabel); bgColorWrap.appendChild(bgColorInput);

    var btnColorRefresh = document.createElement('button');
    btnColorRefresh.textContent = '\u21BB'; // refresh glyph
    btnColorRefresh.title = 'Apply colors';
    btnColorRefresh.style.cssText = 'align-self:flex-end;padding:6px 10px;border:1px solid #1c1c1c;background:#ececea;color:#141414;font-family:inherit;font-size:13px;cursor:pointer;';

    function applyColors(){
      var hexRe = /^#?[0-9a-fA-F]{6}$/;
      if(hexRe.test(arrowColorInput.value)){
        var ac = arrowColorInput.value.replace('#','');
        instanceBaseMat.color.set('#'+ac);
        if(instanceAccentMat === instanceBaseMat) instanceAccentMat.color.set('#'+ac);
      }
      if(hexRe.test(bgColorInput.value)){
        var bc = '#'+bgColorInput.value.replace('#','');
        effectiveBgColor = bc;
        renderer.setClearColor(bc, 1);
        if(scene.fog) scene.fog.color.set(bc);
        if(options.minimalSphere) globeSolidMat.color.set(bc);
      }
    }
    btnColorRefresh.addEventListener('click', applyColors);
    [arrowColorInput, bgColorInput].forEach(function(inp){
      inp.addEventListener('keydown', function(e){ if(e.key === 'Enter') applyColors(); });
    });

    colorsRow.appendChild(arrowColorWrap); colorsRow.appendChild(bgColorWrap); colorsRow.appendChild(btnColorRefresh);
    panelEl.appendChild(colorsRow);

    var divider2 = document.createElement('div');
    divider2.style.cssText = 'border-top:1px solid #1c1c1c;margin:14px 0;';
    panelEl.appendChild(divider2);

    var exportLabel = document.createElement('div');
    exportLabel.textContent = 'EXPORT';
    exportLabel.style.cssText = 'font-weight:700;letter-spacing:0.5px;margin-bottom:8px;';
    panelEl.appendChild(exportLabel);
    var exportRow = document.createElement('div');
    exportRow.style.cssText = 'display:flex;gap:8px;margin-bottom:14px;';
    var btnPng = document.createElement('button');
    btnPng.textContent = 'EXPORT PNG (P)';
    var btnSvg = document.createElement('button');
    btnSvg.textContent = 'EXPORT SVG (S)';
    [btnPng, btnSvg].forEach(function(b){
      b.style.cssText = 'flex:1;padding:8px 4px;border:1px solid #1c1c1c;background:#ececea;color:#141414;font-family:inherit;font-size:10px;letter-spacing:0.4px;cursor:pointer;';
    });
    exportRow.appendChild(btnPng); exportRow.appendChild(btnSvg);
    panelEl.appendChild(exportRow);

    var btnEmbed = document.createElement('button');
    btnEmbed.textContent = 'EXPORT EMBED CODE';
    btnEmbed.style.cssText = 'width:100%;padding:8px 4px;border:1px solid #1c1c1c;background:#ececea;color:#141414;font-family:inherit;font-size:10px;letter-spacing:0.4px;cursor:pointer;margin-bottom:14px;';
    panelEl.appendChild(btnEmbed);

    readoutEl = document.createElement('div');
    readoutEl.style.cssText = 'margin-top:4px;line-height:1.6;color:#3a3a38;';
    panelEl.appendChild(readoutEl);
  }

  var canvasWrap = document.createElement('div');
  canvasWrap.style.cssText = 'flex:1 1 auto;position:relative;background:#0b0b0c;min-width:0;height:100%;';
  root.appendChild(canvasWrap);
  var canvas = document.createElement('canvas');
  canvas.style.cssText = 'display:block;width:100%;height:100%;';
  canvasWrap.appendChild(canvas);

  /* ======================================================================
     THREE SETUP (scoped to this instance)
  ====================================================================== */
  var renderer = new THREE.WebGLRenderer({canvas:canvas, antialias:true});
  renderer.setPixelRatio(Math.min(global.devicePixelRatio||1, 2));
  renderer.setClearColor(effectiveBgColor, 1);

  var scene = new THREE.Scene();
  var camera = new THREE.PerspectiveCamera(45, 1, 0.1, 100);
  var orbit = {radius:6.62, theta:6.223844112611779, phi:1.616174887346749};
  if(initialCamDefault){
    orbit.theta = initialCamDefault.theta*Math.PI/180;
    orbit.phi = initialCamDefault.phi*Math.PI/180;
    orbit.radius = initialCamDefault.radius;
  }
  if(options.camera){
    if(typeof options.camera.theta === 'number') orbit.theta = options.camera.theta;
    if(typeof options.camera.phi === 'number') orbit.phi = options.camera.phi;
    if(typeof options.camera.radius === 'number') orbit.radius = options.camera.radius;
  }
  function applyOrbit(){
    camera.position.x = orbit.radius * Math.sin(orbit.phi) * Math.sin(orbit.theta);
    camera.position.y = orbit.radius * Math.cos(orbit.phi);
    camera.position.z = orbit.radius * Math.sin(orbit.phi) * Math.cos(orbit.theta);
    camera.lookAt(0,0,0);
  }
  applyOrbit();

  function resize(){
    var w = canvasWrap.clientWidth, h = canvasWrap.clientHeight;
    if(w<2||h<2) return;
    renderer.setSize(w,h,false);
    camera.aspect = w/h;
    camera.updateProjectionMatrix();
  }
  var resizeObs = new (global.ResizeObserver || function(){ this.observe=function(){}; this.disconnect=function(){}; })(resize);
  resizeObs.observe(container);
  resize();

  var currentArrowStyle = null;
  var baseMesh = new THREE.InstancedMesh(sharedArrowGeoByStyle[cfg.global.arrowStyle], instanceBaseMat, MAX_INSTANCES);
  var accentMesh = new THREE.InstancedMesh(sharedArrowGeoByStyle[cfg.global.arrowStyle], instanceAccentMat, MAX_INSTANCES);
  currentArrowStyle = cfg.global.arrowStyle;
  function syncArrowStyleGeometry(){
    if(cfg.global.arrowStyle === currentArrowStyle) return;
    currentArrowStyle = cfg.global.arrowStyle;
    var geo = sharedArrowGeoByStyle[currentArrowStyle];
    baseMesh.geometry = geo;
    accentMesh.geometry = geo;
  }
  baseMesh.count = 0; accentMesh.count = 0;
  scene.add(baseMesh, accentMesh);

  // Arrows are fully self-illuminated (MeshBasicMaterial ignores lighting entirely,
  // so every face of the same object is one uniform flat color — no shaded faces).
  // Depth cueing ("farther = grayer") comes from scene fog instead, which fades each
  // fragment toward a neutral gray as it recedes from the camera — applied by distance,
  // not by surface angle, so it never creates a shaded/lit look on the object itself.
  // Fog color follows the same override as the background: with a custom bgColor,
  // distant fragments (e.g. the far side of the GLOBE sphere) fade toward that same
  // color instead of an unrelated gray, so nothing reads as a mismatched seam against
  // a recolored background.
  scene.fog = new THREE.Fog((typeof options.bgColor !== 'undefined') ? options.bgColor : 0x55534f, 3, 15);

  var poleGroup = new THREE.Group();
  scene.add(poleGroup);

  /* ---------------- NETWORK mode meshes: node rectangles + edge lines ---------------- */
  var nodeMesh = new THREE.InstancedMesh(sharedNodeGeo, instanceBaseMat, MAX_NETWORK_NODES);
  var nodeAccentMesh = new THREE.InstancedMesh(sharedNodeGeo, instanceAccentMat, MAX_NETWORK_NODES);
  nodeMesh.count = 0; nodeAccentMesh.count = 0;
  scene.add(nodeMesh, nodeAccentMesh);

  var edgeLines = []; // pre-allocated pool of THREE.Line, reused as edge count changes
  function ensureEdgeLinePool(n){
    while(edgeLines.length < n){
      var geo = new THREE.BufferGeometry();
      geo.setAttribute('position', new THREE.Float32BufferAttribute(new Float32Array((EDGE_SEGMENTS+1)*3), 3));
      var line = new THREE.Line(geo, sharedLineMatBase);
      line.frustumCulled = false;
      scene.add(line);
      edgeLines.push(line);
    }
  }

  /* ---------------- GLOBE mode visuals: the planet + its lat/long grid ---------------- */
  var globeSphereGeo = new THREE.SphereGeometry(1, 32, 24);
  var globeSolidMat = new THREE.MeshBasicMaterial({color: options.minimalSphere ? effectiveBgColor : 0x161613});
  var globeGridMat = new THREE.MeshBasicMaterial({color:0x3c3c38, wireframe:true, transparent:true, opacity:0.5});
  var globeSolidMesh = new THREE.Mesh(globeSphereGeo, globeSolidMat);
  var globeGridMesh = new THREE.Mesh(globeSphereGeo, globeGridMat);
  globeGridMesh.scale.setScalar(1.003);
  globeSolidMesh.visible = false; globeGridMesh.visible = false;
  scene.add(globeSolidMesh, globeGridMesh);
  var showGlobeGrid = !options.minimalSphere; // color-matched-to-background solid sphere still occludes the back hemisphere correctly; the grid overlay is just hidden

  /* ---------------- SPIRAL mode visual: a small centered sphere, same look ---------------- */
  var spiralSolidMesh = new THREE.Mesh(globeSphereGeo, globeSolidMat);
  var spiralGridMesh = new THREE.Mesh(globeSphereGeo, globeGridMat);
  spiralGridMesh.scale.setScalar(1.003);
  spiralSolidMesh.visible = false; spiralGridMesh.visible = false;
  scene.add(spiralSolidMesh, spiralGridMesh);
  function hideSpiralVisuals(){ spiralSolidMesh.visible = false; spiralGridMesh.visible = false; }

  /* ---------------- interaction ---------------- */
  var raycaster = new THREE.Raycaster();
  var groundPlane = new THREE.Plane(new THREE.Vector3(0,0,1), 0);
  var interactionPoint = new THREE.Vector3();
  var interactionActive = false;
  var mouseNDC = new THREE.Vector2();
  var isDragging=false, dragMoved=false, lastX=0, lastY=0;

  function updatePointerNDC(e){
    var rect = canvas.getBoundingClientRect();
    mouseNDC.x = ((e.clientX-rect.left)/rect.width)*2-1;
    mouseNDC.y = -((e.clientY-rect.top)/rect.height)*2+1;
  }
  if(interactiveEnabled){
    canvas.addEventListener('pointerdown', function(e){
      isDragging=true; dragMoved=false; lastX=e.clientX; lastY=e.clientY;
      try{ canvas.setPointerCapture(e.pointerId); }catch(err){}
    });
    canvas.addEventListener('pointermove', function(e){
      updatePointerNDC(e);
      interactionActive = true;
      raycaster.setFromCamera(mouseNDC, camera);
      raycaster.ray.intersectPlane(groundPlane, interactionPoint);
      if(isDragging){
        var dx = e.clientX-lastX, dy = e.clientY-lastY;
        if(Math.abs(dx)+Math.abs(dy) > 3) dragMoved = true;
        if(dragMoved && !cameraLocked){
          orbit.theta -= dx*0.006;
          orbit.phi = Math.min(2.7, Math.max(0.35, orbit.phi - dy*0.006));
        }
        lastX=e.clientX; lastY=e.clientY;
      }
    });
    canvas.addEventListener('pointerup', function(e){
      if(!dragMoved && state.mode==='growth'){
        updatePointerNDC(e);
        raycaster.setFromCamera(mouseNDC, camera);
        var hit = new THREE.Vector3();
        raycaster.ray.intersectPlane(groundPlane, hit);
        resetGrowth(hit);
      }
      isDragging=false; dragMoved=false;
    });
    canvas.addEventListener('pointerleave', function(){ interactionActive=false; });
    canvas.addEventListener('wheel', function(e){
      e.preventDefault();
      orbit.radius = Math.min(20, Math.max(3, orbit.radius*(1+e.deltaY*0.001)));
    }, {passive:false});
  }

  /* ---------------- FIELD MODE ----------------
     Back to the original multi-pole vortex flow (each particle just follows the
     combined tangential field from every pole, plus the cursor) — that IS the
     magnetic-field-lines behavior. The only addition is a lightweight separation
     force so arrows that end up close together push apart instead of overlapping;
     it uses a spatial grid rebuilt once per frame so it stays cheap even at 800
     particles (no O(n^2) scan). */
  var fieldParticles = [], poles = [];
  function computePoles(){
    var n = cfg.field.poles;
    poles = []; poleGroup.clear();
    for(var i=0;i<n;i++){
      var a = (i/n)*Math.PI*2;
      var p = new THREE.Vector3(Math.cos(a)*2.2, Math.sin(a)*2.2*0.6, Math.sin(a*2)*1.0);
      poles.push(p);
      var m = new THREE.Mesh(sharedPoleGeo, sharedPoleMat);
      m.position.copy(p);
      poleGroup.add(m);
    }
  }
  function initField(){
    computePoles();
    var n = cfg.field.count;
    fieldParticles = [];
    for(var i=0;i<n;i++){
      var pole = poles[Math.floor(Math.random()*poles.length)] || new THREE.Vector3();
      var ang = Math.random()*Math.PI*2;
      var r = 0.6 + Math.random()*3.2;
      fieldParticles.push({
        pos: new THREE.Vector3(pole.x+Math.cos(ang)*r, pole.y+Math.sin(ang)*r, (Math.random()-0.5)*1.5),
        vel: new THREE.Vector3(),
        seed: Math.random()*1000
      });
    }
  }
  var tmpRel=new THREE.Vector3(), tmpTan=new THREE.Vector3(), tmpFieldV=new THREE.Vector3();
  function fieldVectorAt(pos){
    tmpFieldV.set(0,0,0);
    for(var i=0;i<poles.length;i++){
      tmpRel.subVectors(pos, poles[i]);
      var d2 = Math.max(tmpRel.lengthSq(), 0.05);
      tmpTan.set(-tmpRel.y, tmpRel.x, 0);
      if(tmpTan.lengthSq()>1e-8) tmpTan.normalize();
      tmpTan.multiplyScalar(cfg.field.strength*3.0/Math.sqrt(d2));
      tmpFieldV.add(tmpTan);
    }
    if(interactionActive && cfg.field.cursorPull>0){
      tmpRel.subVectors(pos, interactionPoint);
      var d2c = Math.max(tmpRel.lengthSq(), 0.05);
      tmpTan.set(-tmpRel.y, tmpRel.x, 0);
      if(tmpTan.lengthSq()>1e-8) tmpTan.normalize();
      tmpTan.multiplyScalar(cfg.field.cursorPull*3.0/Math.sqrt(d2c));
      tmpFieldV.add(tmpTan);
    }
    return tmpFieldV;
  }
  // spatial hash for cheap neighbor lookups (separation only — not used for the field itself)
  var FIELD_SEP_RADIUS = 0.32, FIELD_SEP_STRENGTH = 0.5;
  var fieldGrid = {};
  function rebuildFieldGrid(){
    fieldGrid = {};
    var cell = FIELD_SEP_RADIUS;
    for(var i=0;i<fieldParticles.length;i++){
      var p = fieldParticles[i].pos;
      var key = Math.floor(p.x/cell)+','+Math.floor(p.y/cell);
      var bucket = fieldGrid[key];
      if(!bucket){ bucket = fieldGrid[key] = []; }
      bucket.push(i);
    }
  }
  var fieldSepResult = {x:0,y:0};
  function fieldSeparationAt(idx){
    var p = fieldParticles[idx].pos;
    var cell = FIELD_SEP_RADIUS;
    var cx = Math.floor(p.x/cell), cy = Math.floor(p.y/cell);
    var fx=0, fy=0;
    for(var dx=-1;dx<=1;dx++){
      for(var dy=-1;dy<=1;dy++){
        var bucket = fieldGrid[(cx+dx)+','+(cy+dy)];
        if(!bucket) continue;
        for(var b=0;b<bucket.length;b++){
          var j = bucket[b];
          if(j===idx) continue;
          var q = fieldParticles[j].pos;
          var ddx = p.x-q.x, ddy = p.y-q.y;
          var d2 = ddx*ddx+ddy*ddy;
          if(d2>1e-6 && d2<FIELD_SEP_RADIUS*FIELD_SEP_RADIUS){
            var d = Math.sqrt(d2);
            var f = (FIELD_SEP_RADIUS-d)/FIELD_SEP_RADIUS;
            fx += (ddx/d)*f; fy += (ddy/d)*f;
          }
        }
      }
    }
    fieldSepResult.x = fx*FIELD_SEP_STRENGTH; fieldSepResult.y = fy*FIELD_SEP_STRENGTH;
    return fieldSepResult;
  }
  // Multi-vortex flow fields don't naturally keep passive particles bounded — with
  // several poles interacting, some particles will eventually drift arbitrarily far
  // (this is the real dispersal behavior seen with several interacting vortices).
  // A soft "leash" fixes it without fighting the natural flow at normal working
  // distances: no pull at all until a particle drifts past CONTAINMENT_RADIUS from
  // its nearest pole, then an increasingly firm pull back toward that pole.
  var FIELD_CONTAINMENT_RADIUS = 4.0, FIELD_CONTAINMENT_K = 0.9;
  function fieldContainmentAt(pos){
    var nearest = null, nearestD2 = Infinity;
    for(var i=0;i<poles.length;i++){
      var d2 = pos.distanceToSquared(poles[i]);
      if(d2<nearestD2){ nearestD2=d2; nearest=poles[i]; }
    }
    if(!nearest) return null;
    var d = Math.sqrt(nearestD2);
    if(d <= FIELD_CONTAINMENT_RADIUS) return null;
    var excess = d - FIELD_CONTAINMENT_RADIUS;
    var pull = Math.min(excess*excess*FIELD_CONTAINMENT_K, 6.0); // capped so it never overreacts violently
    tmpRel.subVectors(nearest, pos).normalize().multiplyScalar(pull);
    return tmpRel;
  }
  function updateField(dt, time){
    rebuildFieldGrid();
    for(var i=0;i<fieldParticles.length;i++){
      var pt = fieldParticles[i];
      var f = fieldVectorAt(pt.pos);
      pt.vel.lerp(f, 0.15);
      var sep = fieldSeparationAt(i);
      var leash = fieldContainmentAt(pt.pos);
      pt.pos.addScaledVector(pt.vel, dt*cfg.field.speed);
      pt.pos.x += sep.x*dt;
      pt.pos.y += sep.y*dt;
      if(leash) pt.pos.addScaledVector(leash, dt);
      pt.pos.z += Math.sin(time*0.5+pt.seed)*0.01;
    }
  }

  /* ---------------- SCHOOL MODE ---------------- */
  var agents = [];
  function initSchool(){
    var n = cfg.school.count;
    agents = [];
    for(var i=0;i<n;i++){
      agents.push({
        pos: new THREE.Vector3((Math.random()-0.5)*4,(Math.random()-0.5)*4,(Math.random()-0.5)*2),
        vel: new THREE.Vector3((Math.random()-0.5),(Math.random()-0.5),(Math.random()-0.5)*0.3),
        // most fish swim near full speed; a minority are noticeably slower and trail the pack
        speedFactor: Math.random() < 0.22 ? (0.35 + Math.random()*0.3) : (0.85 + Math.random()*0.15)
      });
    }
  }
  var NEIGHBOR_R=1.6, SEP_R=0.55;
  var schoolAvgVel = new THREE.Vector3(), schoolTravelAxis = new THREE.Vector3(1,0,0);
  function updateSchool(dt){
    var n = agents.length, c = cfg.school;

    // average heading of the whole school — used to stretch the containment shape
    // long along the direction of travel instead of pulling everyone into a sphere.
    schoolAvgVel.set(0,0,0);
    for(var k=0;k<n;k++) schoolAvgVel.add(agents[k].vel);
    if(n>0) schoolAvgVel.divideScalar(n);
    if(schoolAvgVel.lengthSq()>1e-6) schoolTravelAxis.copy(schoolAvgVel).normalize();

    for(var i=0;i<n;i++){
      var a = agents[i];
      var coh=new THREE.Vector3(), ali=new THREE.Vector3(), sep=new THREE.Vector3();
      var count=0;
      for(var j=0;j<n;j++){
        if(i===j) continue;
        var b = agents[j];
        var d2 = a.pos.distanceToSquared(b.pos);
        if(d2 < NEIGHBOR_R*NEIGHBOR_R){
          coh.add(b.pos); ali.add(b.vel); count++;
          if(d2 < SEP_R*SEP_R && d2>1e-6){
            var away = new THREE.Vector3().subVectors(a.pos,b.pos).multiplyScalar(1/d2);
            sep.add(away);
          }
        }
      }
      var force = new THREE.Vector3();
      if(count>0){
        coh.divideScalar(count).sub(a.pos).multiplyScalar(c.cohesion);
        ali.divideScalar(count).sub(a.vel).multiplyScalar(c.alignment);
        force.add(coh).add(ali);
      }
      force.add(sep.multiplyScalar(c.separation));

      // elongated (oval) containment: weak pull along the travel axis (lets the
      // school stretch out lengthwise), stronger pull perpendicular to it (keeps
      // it narrow) — instead of a uniform pull that keeps the shape round.
      var alongAmt = a.pos.dot(schoolTravelAxis);
      var alongVec = schoolTravelAxis.clone().multiplyScalar(alongAmt);
      var perpVec = a.pos.clone().sub(alongVec);
      force.addScaledVector(alongVec, -0.007);
      force.addScaledVector(perpVec, -0.05);

      if(interactionActive && c.cursorPull>0){
        var toC = new THREE.Vector3().subVectors(interactionPoint, a.pos).multiplyScalar(c.cursorPull*0.4);
        force.add(toC);
      }
      a.vel.addScaledVector(force, dt);
      var maxSpeed = 2.2 * a.speedFactor;
      if(a.vel.length()>maxSpeed) a.vel.setLength(maxSpeed);
      a.pos.addScaledVector(a.vel, dt*c.speed);
    }
  }

  /* ---------------- GROWTH MODE ---------------- */
  var segments=[], activeTwigs=[], growthAccum=0, resetPending=false, resetTimer=0;
  var lastSeed = new THREE.Vector3(0,-2.6,0);
  var AXIS_POOL=[];
  (function(){ for(var i=0;i<24;i++) AXIS_POOL.push(new THREE.Vector3(Math.random()-0.5,Math.random()-0.5,Math.random()-0.5).normalize()); })();

  function resetGrowth(seedPos){
    if(seedPos) lastSeed.copy(seedPos);
    segments = [];
    activeTwigs = [{pos:lastSeed.clone(), dir:new THREE.Vector3(0,1,0), len:0.9, depth:0}];
    resetPending=false; resetTimer=0; growthAccum=0;
  }
  function growthStep(time, interval){
    var c = cfg.growth;
    var newTwigs = [];
    for(var t=0;t<activeTwigs.length;t++){
      var twig = activeTwigs[t];
      var dir = twig.dir.clone();
      if(interactionActive && c.cursorPull>0){
        var toC = new THREE.Vector3().subVectors(interactionPoint, twig.pos);
        if(toC.lengthSq()>1e-6) toC.normalize();
        dir.lerp(toC, Math.min(0.9,0.15*c.cursorPull)).normalize();
      }
      dir.x += (Math.random()-0.5)*c.variance*0.6;
      dir.y += (Math.random()-0.5)*c.variance*0.6;
      dir.z += (Math.random()-0.5)*c.variance*0.3;
      if(dir.lengthSq()<1e-8) dir.set(0,1,0);
      dir.normalize();
      var len = twig.len;
      var newPos = twig.pos.clone().addScaledVector(dir, len);
      segments.push({pos:twig.pos.clone(), dir:dir.clone(), len:len, birth:time, growDur:interval});
      var nd = twig.depth+1;
      if(nd < c.maxDepth && segments.length < c.maxSegments){
        if(Math.random() < c.spawnChance){
          for(var b=0;b<2;b++){
            var bdir = dir.clone();
            var ang = (c.branchAngle*Math.PI/180) * (b===0?1:-1);
            var axis = AXIS_POOL[Math.floor(Math.random()*AXIS_POOL.length)];
            bdir.applyAxisAngle(axis, ang);
            newTwigs.push({pos:newPos.clone(), dir:bdir.normalize(), len:len*0.78, depth:nd});
          }
        } else {
          newTwigs.push({pos:newPos.clone(), dir:dir, len:len*0.85, depth:nd});
        }
      }
    }
    activeTwigs = newTwigs;
    if(activeTwigs.length===0 || segments.length>=c.maxSegments){
      resetPending=true; resetTimer=10.0; // pause once fully grown, before regrowing
    }
  }
  function updateGrowth(dt, time){
    if(resetPending){
      resetTimer -= dt;
      if(resetTimer<=0) resetGrowth();
      return;
    }
    growthAccum += dt;
    var interval = 0.5/Math.max(0.05,cfg.growth.growSpeed);
    var guard=0;
    while(growthAccum > interval && guard < 20){
      growthAccum -= interval;
      growthStep(time, interval);
      guard++;
      if(resetPending) break;
    }
  }
  resetGrowth();

  /* ---------------- NETWORK MODE ---------------- */
  var networkNodes = [], networkEdges = [];
  function initNetwork(){
    var n = cfg.network.count;
    networkNodes = [];
    for(var i=0;i<n;i++){
      networkNodes.push({
        pos: new THREE.Vector3((Math.random()-0.5)*5,(Math.random()-0.5)*3.5,(Math.random()-0.5)*2.5),
        vel: new THREE.Vector3(),
        w: 0.9 + Math.random()*2.6,    // rectangle width — wide, horizontal bars
        h: 0.35 + Math.random()*0.5,   // rectangle height — kept short
        swaySeed: Math.random()*1000
      });
    }
    // fixed connection topology: each node links to a few others, chosen once so the
    // graph doesn't rewire itself every frame — only the node positions drift.
    networkEdges = [];
    var perNode = Math.round(cfg.network.edgesPerNode);
    for(var a=0;a<n && networkEdges.length<MAX_NETWORK_EDGES;a++){
      for(var k=0;k<perNode;k++){
        var b = Math.floor(Math.random()*n);
        if(b===a) continue;
        networkEdges.push({a:a, b:b, curveSeed:Math.random()*1000, curveSign:(Math.random()<0.5?-1:1)});
        if(networkEdges.length>=MAX_NETWORK_EDGES) break;
      }
    }
  }
  var NET_NEIGHBOR_R = 2.2;
  function updateNetwork(dt, time){
    var n = networkNodes.length, c = cfg.network;
    for(var i=0;i<n;i++){
      var a = networkNodes[i];
      var coh = new THREE.Vector3();
      var count = 0;
      for(var j=0;j<n;j++){
        if(i===j) continue;
        var b = networkNodes[j];
        var d2 = a.pos.distanceToSquared(b.pos);
        if(d2 < NET_NEIGHBOR_R*NET_NEIGHBOR_R){
          if(d2 > 0.6*0.6){ coh.add(b.pos); count++; }
          else {
            var away = new THREE.Vector3().subVectors(a.pos,b.pos).multiplyScalar(1/Math.max(d2,0.05));
            a.vel.addScaledVector(away, dt*0.8);
          }
        }
      }
      var force = new THREE.Vector3();
      if(count>0){ coh.divideScalar(count).sub(a.pos).multiplyScalar(c.cohesion); force.add(coh); }
      force.addScaledVector(a.pos, -0.015); // gentle containment so the graph doesn't drift off
      if(interactionActive && c.cursorPull>0){
        var toC = new THREE.Vector3().subVectors(interactionPoint, a.pos).multiplyScalar(c.cursorPull*0.3);
        force.add(toC);
      }
      a.vel.addScaledVector(force, dt);
      a.vel.multiplyScalar(0.96); // damping — a floating drift, not a swarm dash
      a.pos.addScaledVector(a.vel, dt);
      // slow independent sway per axis, on top of the physics drift, for a "living" feel
      a.pos.x += Math.sin(time*c.swaySpeed + a.swaySeed)*0.003*c.sway;
      a.pos.y += Math.cos(time*c.swaySpeed*0.8 + a.swaySeed*1.3)*0.003*c.sway;
    }
  }
  function resetNetwork(){ initNetwork(); }

  /* ---------------- GLOBE MODE ----------------
     Arrows constrained to the surface of a sphere, driven by several tangential
     "vortex" fields (the same cross-product trick as FIELD mode, but computed on the
     sphere so the flow always stays on the surface) plus a slow rigid spin of the
     whole field pattern and a little per-particle turbulence, for a
     magnetic-field-lines-on-a-slowly-turning-planet feel. */
  var globeParticles = [], globePoles = [];
  var GLOBE_SPIN_AXIS = new THREE.Vector3(0.15, 1, 0).normalize();
  function globeSphereCenter(out){
    var c = cfg.globe;
    out.set(c.offsetX||0, -c.sphereSize*c.horizon, 0);
    return out;
  }
  function randomCapDirection(halfAngleDeg){
    var halfAngleRad = halfAngleDeg*Math.PI/180;
    var cosT = 1 - Math.random()*(1-Math.cos(halfAngleRad));
    var theta = Math.acos(cosT);
    var phi = Math.random()*Math.PI*2;
    return new THREE.Vector3(Math.sin(theta)*Math.cos(phi), cosT, Math.sin(theta)*Math.sin(phi));
  }
  function initGlobe(){
    var c = cfg.globe;
    globePoles = [];
    for(var i=0;i<c.poles;i++){
      globePoles.push({dir: randomCapDirection(c.capAngle*0.6), sign: Math.random()<0.5?-1:1});
    }
    var center = globeSphereCenter(new THREE.Vector3());
    globeParticles = [];
    for(var j=0;j<c.count;j++){
      var dir = randomCapDirection(c.capAngle*0.85);
      globeParticles.push({
        pos: center.clone().addScaledVector(dir, c.sphereSize),
        vel: new THREE.Vector3(),
        seed: Math.random()*1000,
        axisSeed: new THREE.Vector3(Math.random()-0.5,Math.random()-0.5,Math.random()-0.5).normalize()
      });
    }
  }
  function resetGlobe(){ initGlobe(); }

  /* ---------------- STARTREK MODE ----------------
     A classic warp/starfield flythrough: no vortices, no attraction — arrows just
     travel in one constant direction (the camera's forward axis, fixed once so a
     later camera drag doesn't reshuffle the tunnel) and recycle to the far distance
     once they pass the camera. A star lined up with the view axis points straight
     back at the camera, so with the CONE arrow style it reads as a simple point —
     exactly like a distant star — while off-axis ones streak past at an angle,
     purely from perspective, the same way real starfield/warp effects work. */
  var starParticles = [];
  var starForward = new THREE.Vector3(), starRight = new THREE.Vector3(), starUp = new THREE.Vector3();
  var STAR_FAR = 26, STAR_NEAR = 0.8, STAR_LEN = 0.45;
  function computeStarBasis(){
    starForward.copy(camera.position).normalize().multiplyScalar(-1); // camera -> origin
    var worldUp = new THREE.Vector3(0,1,0);
    starRight.crossVectors(starForward, worldUp);
    if(starRight.lengthSq()<1e-6) starRight.set(1,0,0); else starRight.normalize();
    starUp.crossVectors(starRight, starForward).normalize();
  }
  function initStarTrek(){
    computeStarBasis();
    starParticles = [];
    var c = cfg.startrek;
    for(var i=0;i<c.count;i++){
      starParticles.push({
        depth: STAR_NEAR + Math.random()*(STAR_FAR-STAR_NEAR),
        offR: (Math.random()-0.5)*2*c.spread,
        offU: (Math.random()-0.5)*2*c.spread
      });
    }
  }
  function resetStarTrek(){ initStarTrek(); }
  function updateStarTrek(dt){
    var c = cfg.startrek;
    var steerR=0, steerU=0;
    if(interactionActive && c.cursorPull>0){
      steerR = interactionPoint.dot(starRight)*0.22*c.cursorPull;
      steerU = interactionPoint.dot(starUp)*0.22*c.cursorPull;
    }
    for(var i=0;i<starParticles.length;i++){
      var s = starParticles[i];
      s.depth -= c.speed*dt*4.0;
      s.offR += steerR*dt;
      s.offU += steerU*dt;
      if(s.depth < STAR_NEAR){
        s.depth = STAR_FAR;
        s.offR = (Math.random()-0.5)*2*c.spread;
        s.offU = (Math.random()-0.5)*2*c.spread;
      }
    }
  }

  /* ---------------- SPIRAL MODE ----------------
     Pure surface flow — no pull toward any hidden "core", nothing happens off the
     visible front. Two poles (one per region, opposite spin) drive the circulation;
     TURBULENCE (per-particle phase, same technique as GLOBE) is what bends that into
     organic non-circular wandering rather than a perfect ring. This mirrors GLOBE's
     field/containment math closely, including its velocity cap. An earlier version
     used two close same-sign poles per region to try to bend the shape geometrically
     — that created an over-attracting "channel" between the two regions that pulled
     the whole population onto one thin curve within a few seconds; back to one clean
     pole per region fixes that. */
  var spiralParticles = [], spiralPoles = [];
  var SPIRAL_FRONT = new THREE.Vector3(0,0,1); // the camera-facing axis for this mode's fixed front-on view
  var SPIRAL_MAX_SPEED = 2.2;
  function poleDirFromPolar(polarDeg, azimuthDeg){
    var polar = polarDeg*Math.PI/180, az = azimuthDeg*Math.PI/180;
    return new THREE.Vector3(Math.sin(polar)*Math.cos(az), Math.sin(polar)*Math.sin(az), Math.cos(polar));
  }
  function initSpiral(){
    // Two regions (front-hemisphere-safe, ~40-50deg from the view axis), each made of
    // two close poles spinning the same way — that pairing is what bends the flow into
    // organic nested loops instead of one clean circle per region.
    // Two poles (not four) — two same-sign poles placed close together turned out to
    // create an over-attracting "channel" between the regions that pulls the whole
    // population onto one thin curve within a few seconds. A single pole per region,
    // same layout that's already proven stable in FIELD/GLOBE, plus TURBULENCE below
    // for the organic non-circular bending.
    spiralPoles = [
      {dir: poleDirFromPolar(44, 100), sign: 1},
      {dir: poleDirFromPolar(44, 280), sign:-1}
    ];
    spiralParticles = [];
    var c = cfg.spiral;
    for(var i=0;i<c.count;i++){
      var polar = 10+Math.random()*35, az = Math.random()*360;
      spiralParticles.push({
        dirOnSphere: poleDirFromPolar(polar, az),
        vel: new THREE.Vector3(),
        prevDir: null,
        axisSeed: new THREE.Vector3(Math.random()-0.5,Math.random()-0.5,Math.random()-0.5).normalize(),
        seed: Math.random()*1000
      });
    }
  }
  function resetSpiral(){ initSpiral(); }
  var spTmpTan=new THREE.Vector3(), spTmpField=new THREE.Vector3(), spTmpToFront=new THREE.Vector3();
  var spiralClock = 0;
  var spTmpAxis = new THREE.Vector3();
  function spiralTurbulenceAt(dir, time, scale, out){
    if(scale<=0){ out.set(0,0,0); return out; }
    // Large-scale, position-based (not per-particle-phase) — nearby particles get a
    // similar push, so it reads as one smooth, coherent bend in the flow rather than
    // independent jitter.
    var n1 = Math.sin(dir.x*1.3+time*0.15) + Math.sin(dir.y*1.7-time*0.11) + Math.sin(dir.z*1.1+time*0.09);
    var n2 = Math.cos(dir.y*1.5-time*0.13) + Math.cos(dir.z*1.9+time*0.10) + Math.cos(dir.x*1.2-time*0.08);
    spTmpAxis.set(n1, n2, (n1-n2)*0.5);
    if(spTmpAxis.lengthSq()<1e-8) spTmpAxis.set(1,0,0); else spTmpAxis.normalize();
    out.crossVectors(spTmpAxis, dir);
    if(out.lengthSq()>1e-8) out.normalize();
    return out.multiplyScalar(scale*0.35);
  }
  function spiralFieldAt(dirOnSphere, particle, c, time){
    spTmpField.set(0,0,0);
    for(var i=0;i<spiralPoles.length;i++){
      var pole = spiralPoles[i];
      var angDist = Math.max(1 - dirOnSphere.dot(pole.dir), 0.02);
      spTmpTan.crossVectors(pole.dir, dirOnSphere);
      if(spTmpTan.lengthSq()>1e-8) spTmpTan.normalize();
      spTmpTan.multiplyScalar(pole.sign*c.strength*0.6/angDist);
      spTmpField.add(spTmpTan);
    }
    spTmpField.add(spiralTurbulenceAt(dirOnSphere, time, c.turbulence, spTmpTan.clone()));
    return spTmpField;
  }
  function updateSpiral(dt, time){
    spiralClock = time;
    var c = cfg.spiral;
    var capRad = c.capAngle*Math.PI/180;
    var hardCapRad = capRad*1.1;
    for(var i=0;i<spiralParticles.length;i++){
      var p = spiralParticles[i];
      var dir = p.dirOnSphere;
      var field = spiralFieldAt(dir, p, c);

      var angFromFront = Math.acos(Math.min(1, Math.max(-1, dir.dot(SPIRAL_FRONT))));
      if(angFromFront > capRad){
        spTmpToFront.copy(SPIRAL_FRONT).addScaledVector(dir, -dir.dot(SPIRAL_FRONT));
        if(spTmpToFront.lengthSq()>1e-8) spTmpToFront.normalize();
        var excess = angFromFront-capRad;
        field.addScaledVector(spTmpToFront, Math.min(excess*excess*14, 6));
      }

      p.vel.lerp(field, 0.15);
      p.vel.addScaledVector(dir, -dir.dot(p.vel)); // keep tangent
      if(p.vel.lengthSq() > SPIRAL_MAX_SPEED*SPIRAL_MAX_SPEED) p.vel.setLength(SPIRAL_MAX_SPEED);

      p.prevDir = dir.clone();
      dir.addScaledVector(p.vel, dt*c.speed*0.35).normalize();

      var angNow = Math.acos(Math.min(1, Math.max(-1, dir.dot(SPIRAL_FRONT))));
      if(angNow > hardCapRad){
        var axis = new THREE.Vector3().crossVectors(SPIRAL_FRONT, dir);
        if(axis.lengthSq()>1e-10){
          axis.normalize();
          dir.copy(SPIRAL_FRONT).applyAxisAngle(axis, hardCapRad);
        }
      }
    }
  }

  var gTmpRel=new THREE.Vector3(), gTmpTan=new THREE.Vector3(), gTmpField=new THREE.Vector3(), gTmpCenter=new THREE.Vector3();
  var GLOBE_UP = new THREE.Vector3(0,1,0);
  function globeFieldAt(dirOnSphere, particle, c){
    gTmpField.set(0,0,0);
    for(var i=0;i<globePoles.length;i++){
      var pole = globePoles[i];
      var angDist = Math.max(1 - dirOnSphere.dot(pole.dir), 0.02); // 0 at pole, up to 2 at antipode
      gTmpTan.crossVectors(pole.dir, dirOnSphere); // automatically tangent to sphere at dirOnSphere
      if(gTmpTan.lengthSq()>1e-8) gTmpTan.normalize();
      gTmpTan.multiplyScalar(pole.sign*c.strength*0.6/angDist);
      gTmpField.add(gTmpTan);
    }
    if(c.turbulence>0){
      gTmpTan.crossVectors(particle.axisSeed, dirOnSphere);
      if(gTmpTan.lengthSq()>1e-8) gTmpTan.normalize();
      gTmpTan.multiplyScalar(Math.sin(globeClock*0.4 + particle.seed)*c.turbulence*0.35);
      gTmpField.add(gTmpTan);
    }
    return gTmpField;
  }
  var globeClock = 0;
  var GLOBE_MAX_SPEED = 2.2;
  function updateGlobe(dt, time){
    globeClock = time;
    var c = cfg.globe;
    globeSphereCenter(gTmpCenter);
    // slowly spin the whole pole configuration, like weather patterns riding a turning planet
    var spinAngle = c.spin*0.15*dt;
    if(spinAngle !== 0){
      for(var p=0;p<globePoles.length;p++) globePoles[p].dir.applyAxisAngle(GLOBE_SPIN_AXIS, spinAngle);
    }
    var capLimitRad = c.capAngle*Math.PI/180;
    for(var i=0;i<globeParticles.length;i++){
      var particle = globeParticles[i];
      gTmpRel.subVectors(particle.pos, gTmpCenter);
      var dirOnSphere = gTmpRel.normalize();
      var field = globeFieldAt(dirOnSphere, particle, c);
      if(interactionActive && c.cursorPull>0){
        var cursorRel = new THREE.Vector3().subVectors(interactionPoint, gTmpCenter);
        if(cursorRel.lengthSq()>1e-6){
          var cursorDir = cursorRel.normalize();
          var ang = Math.max(1 - dirOnSphere.dot(cursorDir), 0.02);
          var cTan = new THREE.Vector3().crossVectors(cursorDir, dirOnSphere);
          if(cTan.lengthSq()>1e-8) cTan.normalize();
          field.add(cTan.multiplyScalar(c.cursorPull*0.6/ang));
        }
      }
      // same "runaway multi-vortex" problem as FIELD — a soft leash back toward the
      // visible cap, so nothing ever slips over the horizon or off-screen, and any
      // sudden speed-up (chaotic advection near where poles' fields cancel) settles
      // back down instead of accelerating away.
      var angFromUp = Math.acos(Math.min(1, Math.max(-1, dirOnSphere.dot(GLOBE_UP))));
      if(angFromUp > capLimitRad){
        var toUp = new THREE.Vector3().copy(GLOBE_UP).addScaledVector(dirOnSphere, -dirOnSphere.dot(GLOBE_UP));
        if(toUp.lengthSq()>1e-8) toUp.normalize();
        var excess = angFromUp - capLimitRad;
        field.addScaledVector(toUp, Math.min(excess*excess*14.0, 6.0));
      }

      particle.vel.lerp(field, 0.15);
      particle.vel.addScaledVector(dirOnSphere, -particle.vel.dot(dirOnSphere)); // keep tangent
      if(particle.vel.lengthSq() > GLOBE_MAX_SPEED*GLOBE_MAX_SPEED) particle.vel.setLength(GLOBE_MAX_SPEED);
      dirOnSphere.addScaledVector(particle.vel, dt*c.speed*0.35).normalize();
      // hard safety clamp: guarantees nothing ever crosses more than ~10% past the
      // visible cap, even during a strong chaotic-advection speed spike.
      var angNow = Math.acos(Math.min(1, Math.max(-1, dirOnSphere.dot(GLOBE_UP))));
      var hardCapRad = capLimitRad*1.1;
      if(angNow > hardCapRad){
        var clampAxis = new THREE.Vector3().crossVectors(GLOBE_UP, dirOnSphere);
        if(clampAxis.lengthSq()>1e-10){
          clampAxis.normalize();
          dirOnSphere = GLOBE_UP.clone().applyAxisAngle(clampAxis, hardCapRad);
        }
      }
      particle.pos.copy(gTmpCenter).addScaledVector(dirOnSphere, c.sphereSize);
    }
  }

  /* ---------------- compose instances ---------------- */
  var X_AXIS=new THREE.Vector3(1,0,0), tmpQuat=new THREE.Quaternion(), tmpScale=new THREE.Vector3(), tmpMat=new THREE.Matrix4();
  var frameArrows = [];
  function assignInstances(list, getPDL){
    frameArrows.length = 0;
    syncArrowStyleGeometry();
    var isFlat = !!FLAT_STYLE_KEYS[cfg.global.arrowStyle];
    var bi=0, ai=0;
    var accentR2 = cfg.global.accentRadius*cfg.global.accentRadius;
    // FLAT/CHEVRON have no separate thickness control — their local width (sw/hw) is
    // authored as a fraction of their own local length, so width must scale by the
    // SAME factor as length (len) to keep the authored proportions exact. CONE keeps
    // its own independent thickness control (arrowScale*lineThickness), unrelated to len.
    var coneWidthScale = cfg.global.arrowScale*cfg.global.lineThickness;
    for(var i=0;i<list.length;i++){
      var pdl = getPDL(list[i]);
      if(!pdl) continue;
      var dir=pdl.dir, len=pdl.len*cfg.global.arrowScale;
      var widthScale = isFlat ? len : coneWidthScale;
      renderPosScratch.copy(pdl.pos);
      if(flatMode) renderPosScratch.z *= FLAT_Z_SQUASH;
      var pos = renderPosScratch;
      if(dir.lengthSq()<1e-8) dir = X_AXIS;
      var isAccent = interactionActive && pos.distanceToSquared(interactionPoint) < accentR2;
      if(isFlat){ computeFlatOrientation(pos, dir, camera.position, tmpQuat); }
      else { tmpQuat.setFromUnitVectors(X_AXIS, dir); }
      tmpScale.set(len, widthScale, widthScale);
      tmpMat.compose(pos, tmpQuat, tmpScale);
      if(isAccent){ if(ai<MAX_INSTANCES){ accentMesh.setMatrixAt(ai++, tmpMat); } }
      else { if(bi<MAX_INSTANCES){ baseMesh.setMatrixAt(bi++, tmpMat); } }
      if(frameArrows.length < MAX_INSTANCES*2) frameArrows.push({pos:pos.clone(), dir:dir.clone(), len:len, accent:isAccent});
    }
    baseMesh.count = bi; accentMesh.count = ai;
    baseMesh.instanceMatrix.needsUpdate = true;
    accentMesh.instanceMatrix.needsUpdate = true;
  }
  var FIELD_LEN=0.4, SCHOOL_LEN=0.34;
  function composeForMode(time){
    if(state.mode==='field'){
      poleGroup.visible = false;
      hideNetworkVisuals();
      hideGlobeVisuals();
      hideSpiralVisuals();
      for(var pgi=0; pgi<poleGroup.children.length; pgi++){
        var poleWorldZ = poles[pgi] ? poles[pgi].z : 0;
        poleGroup.children[pgi].position.z = flatMode ? poleWorldZ*FLAT_Z_SQUASH : poleWorldZ;
      }
      assignInstances(fieldParticles, function(p){
        var d = p.vel.lengthSq()>1e-6 ? p.vel.clone().normalize() : X_AXIS;
        return {pos:p.pos, dir:d, len:FIELD_LEN};
      });
    } else if(state.mode==='school'){
      poleGroup.visible = false;
      hideNetworkVisuals();
      hideGlobeVisuals();
      hideSpiralVisuals();
      assignInstances(agents, function(a){
        var d = a.vel.lengthSq()>1e-6 ? a.vel.clone().normalize() : X_AXIS;
        return {pos:a.pos, dir:d, len:SCHOOL_LEN};
      });
    } else if(state.mode==='growth'){
      poleGroup.visible = false;
      hideNetworkVisuals();
      hideGlobeVisuals();
      hideSpiralVisuals();
      assignInstances(segments, function(s){
        // animate each segment growing outward from its own base point (like an
        // extending stick) instead of popping in at full length instantly.
        var age = Math.max(0, time - s.birth);
        var t = s.growDur>0 ? Math.min(1, age/s.growDur) : 1;
        var curLen = s.len * t;
        if(curLen <= 0.001) return null;
        var center = s.pos.clone().addScaledVector(s.dir, curLen*0.5);
        return {pos:center, dir:s.dir, len:curLen};
      });
    } else if(state.mode==='network'){
      poleGroup.visible = false;
      hideGlobeVisuals();
      hideSpiralVisuals();
      baseMesh.count = 0; accentMesh.count = 0;
      baseMesh.instanceMatrix.needsUpdate = true; accentMesh.instanceMatrix.needsUpdate = true;
      frameArrows.length = 0; // SVG export (arrow-silhouette based) has nothing to draw in this mode yet
      renderNetworkFrame(time);
    } else if(state.mode==='globe'){
      poleGroup.visible = false;
      hideNetworkVisuals();
      hideSpiralVisuals();
      globeSolidMesh.visible = true; globeGridMesh.visible = showGlobeGrid;
      var gc = globeSphereCenter(new THREE.Vector3());
      if(flatMode) gc.z *= FLAT_Z_SQUASH;
      globeSolidMesh.position.copy(gc); globeGridMesh.position.copy(gc);
      globeSolidMesh.scale.setScalar(cfg.globe.sphereSize);
      globeGridMesh.scale.setScalar(cfg.globe.sphereSize*1.003);
      assignInstances(globeParticles, function(p){
        var d = p.vel.lengthSq()>1e-6 ? p.vel.clone().normalize() : X_AXIS;
        return {pos:p.pos, dir:d, len:FIELD_LEN};
      });
    } else if(state.mode==='startrek'){
      poleGroup.visible = false;
      hideNetworkVisuals();
      hideGlobeVisuals();
      hideSpiralVisuals();
      var starDir = starForward.clone().multiplyScalar(-1); // points back toward camera
      assignInstances(starParticles, function(s){
        var pos = camera.position.clone()
          .addScaledVector(starForward, s.depth)
          .addScaledVector(starRight, s.offR)
          .addScaledVector(starUp, s.offU);
        return {pos: pos, dir: starDir, len: STAR_LEN};
      });
    } else if(state.mode==='spiral'){
      poleGroup.visible = false;
      hideNetworkVisuals();
      hideGlobeVisuals();
      spiralSolidMesh.visible = true; spiralGridMesh.visible = true;
      spiralSolidMesh.scale.setScalar(cfg.spiral.sphereSize);
      spiralGridMesh.scale.setScalar(cfg.spiral.sphereSize*1.003);
      assignInstances(spiralParticles, function(p){
        var d = X_AXIS;
        if(p.prevDir){
          d = p.dirOnSphere.clone().sub(p.prevDir);
          if(d.lengthSq()<1e-10) d = X_AXIS; else d.normalize();
        }
        return {pos: p.dirOnSphere.clone().multiplyScalar(cfg.spiral.sphereSize), dir:d, len:FIELD_LEN};
      });
    }
  }
  function hideGlobeVisuals(){
    globeSolidMesh.visible = false; globeGridMesh.visible = false;
  }
  function hideNetworkVisuals(){
    nodeMesh.count = 0; nodeAccentMesh.count = 0;
    nodeMesh.instanceMatrix.needsUpdate = true; nodeAccentMesh.instanceMatrix.needsUpdate = true;
    for(var i=0;i<edgeLines.length;i++) edgeLines[i].visible = false;
  }
  var netTmpQuat = new THREE.Quaternion(), netTmpScale = new THREE.Vector3();
  var netTmpMat = new THREE.Matrix4(), netTmpMid = new THREE.Vector3(), netTmpPerp = new THREE.Vector3();
  var netRenderPosA = new THREE.Vector3(), netRenderPosB = new THREE.Vector3();
  function renderNetworkFrame(time){
    ensureEdgeLinePool(networkEdges.length);
    var accentR2 = cfg.global.accentRadius*cfg.global.accentRadius;
    var nodeSize = cfg.network.nodeSize;
    var ni=0, ai=0;
    for(var i=0;i<networkNodes.length;i++){
      var node = networkNodes[i];
      renderPosScratch.copy(node.pos);
      if(flatMode) renderPosScratch.z *= FLAT_Z_SQUASH;
      var isAccent = interactionActive && renderPosScratch.distanceToSquared(interactionPoint) < accentR2;
      computeSpriteOrientation(renderPosScratch, camera.position, netTmpQuat);
      netTmpScale.set(node.w*nodeSize, node.h*nodeSize, 1);
      netTmpMat.compose(renderPosScratch, netTmpQuat, netTmpScale);
      if(isAccent){ if(ai<MAX_NETWORK_NODES) nodeAccentMesh.setMatrixAt(ai++, netTmpMat); }
      else { if(ni<MAX_NETWORK_NODES) nodeMesh.setMatrixAt(ni++, netTmpMat); }
    }
    nodeMesh.count = ni; nodeAccentMesh.count = ai;
    nodeMesh.instanceMatrix.needsUpdate = true; nodeAccentMesh.instanceMatrix.needsUpdate = true;

    for(var e=0;e<networkEdges.length;e++){
      var edge = networkEdges[e];
      var line = edgeLines[e];
      var rawA = networkNodes[edge.a] ? networkNodes[edge.a].pos : null;
      var rawB = networkNodes[edge.b] ? networkNodes[edge.b].pos : null;
      if(!rawA || !rawB){ line.visible = false; continue; }
      line.visible = true;
      netRenderPosA.copy(rawA); netRenderPosB.copy(rawB);
      if(flatMode){ netRenderPosA.z *= FLAT_Z_SQUASH; netRenderPosB.z *= FLAT_Z_SQUASH; }
      var pa = netRenderPosA, pb = netRenderPosB;
      netTmpMid.addVectors(pa, pb).multiplyScalar(0.5);
      netTmpPerp.subVectors(pb, pa);
      var segLen = netTmpPerp.length();
      netTmpPerp.set(-netTmpPerp.y, netTmpPerp.x, netTmpPerp.z*0.4);
      if(netTmpPerp.lengthSq()>1e-8) netTmpPerp.normalize();
      var bulge = (0.15 + segLen*0.12) * cfg.network.curviness * edge.curveSign;
      bulge += Math.sin(time*0.5 + edge.curveSeed) * 0.05 * cfg.network.curviness; // gentle breathing
      var ctrl = netTmpMid.clone().addScaledVector(netTmpPerp, bulge);

      var posAttr = line.geometry.attributes.position;
      for(var s=0;s<=EDGE_SEGMENTS;s++){
        var t = s/EDGE_SEGMENTS;
        var mt = 1-t;
        var x = mt*mt*pa.x + 2*mt*t*ctrl.x + t*t*pb.x;
        var y = mt*mt*pa.y + 2*mt*t*ctrl.y + t*t*pb.y;
        var z = mt*mt*pa.z + 2*mt*t*ctrl.z + t*t*pb.z;
        posAttr.setXYZ(s, x, y, z);
      }
      posAttr.needsUpdate = true;
      var edgeAccent = interactionActive &&
        (pa.distanceToSquared(interactionPoint) < accentR2 || pb.distanceToSquared(interactionPoint) < accentR2);
      line.material = edgeAccent ? sharedLineMatAccent : sharedLineMatBase;
    }
    for(var extra=networkEdges.length; extra<edgeLines.length; extra++) edgeLines[extra].visible = false;
  }

  function resetSimForMode(){
    if(state.mode==='field') initField();
    else if(state.mode==='school') initSchool();
    else if(state.mode==='growth') resetGrowth();
    else if(state.mode==='network') resetNetwork();
    else if(state.mode==='globe') resetGlobe();
    else if(state.mode==='startrek') resetStarTrek();
    else if(state.mode==='spiral') resetSpiral();
  }
  initField(); initSchool(); initNetwork(); initGlobe(); initStarTrek(); initSpiral();

  /* ---------------- panel: mode params rebuild (needs functions above defined first) ---------------- */
  function rebuildModeParams(){
    if(!showPanel) return;
    modeParamsEl.innerHTML = '';
    modeSectionLabel.textContent = MODES[state.mode].label;
    MODES[state.mode].params.forEach(function(p){
      modeParamsEl.appendChild(makeRow(p,
        function(){ return cfg[state.mode][p.key]; },
        function(v){ cfg[state.mode][p.key] = v; },
        function(){
          if(state.mode==='field' && (p.key==='count'||p.key==='poles')) initField();
          if(state.mode==='school' && p.key==='count') initSchool();
          if(state.mode==='globe' && (p.key==='count'||p.key==='poles')) initGlobe();
          if(state.mode==='spiral' && p.key==='count') initSpiral();
        }
      ));
    });
  }
  if(showPanel) rebuildModeParams();

  /* ---------------- export ---------------- */
  function downloadBlob(content, mime, filename){
    var blob = (content instanceof Blob) ? content : new Blob([content], {type:mime});
    var url = URL.createObjectURL(blob);
    var a = document.createElement('a');
    a.href = url; a.download = filename;
    document.body.appendChild(a); a.click();
    document.body.removeChild(a);
    setTimeout(function(){ URL.revokeObjectURL(url); }, 2000);
  }
  if(showPanel){
    btnPng.addEventListener('click', function(){
      // Export at a higher internal resolution than the live on-screen canvas,
      // independent of window size / devicePixelRatio cap: temporarily grow the
      // renderer's pixel buffer (setSize(..., false) leaves the CSS/display size
      // untouched, so nothing visibly jumps), render once, capture, then restore.
      var curSize = new THREE.Vector2();
      renderer.getSize(curSize);
      var curPixelRatio = renderer.getPixelRatio();
      var EXPORT_MAX_DIM = 3840; // long edge of the exported PNG, in pixels
      var liveLongEdge = Math.max(curSize.x, curSize.y) * curPixelRatio;
      var exportScale = Math.max(1, EXPORT_MAX_DIM / liveLongEdge);

      renderer.setPixelRatio(curPixelRatio * exportScale);
      renderer.setSize(curSize.x, curSize.y, false);
      renderer.render(scene, camera);
      canvas.toBlob(function(blob){
        downloadBlob(blob, 'image/png', 'superplane-'+state.mode+'.png');
        renderer.setPixelRatio(curPixelRatio);
        renderer.setSize(curSize.x, curSize.y, false);
        renderer.render(scene, camera);
      }, 'image/png');
    });
    btnSvg.addEventListener('click', function(){
      var w = renderer.domElement.width, h = renderer.domElement.height;
      var baseParts=[], accentParts=[];
      var isFlat = !!FLAT_STYLE_KEYS[cfg.global.arrowStyle];
      var coneWidthScale = cfg.global.arrowScale*cfg.global.lineThickness;
      var localPoints = ARROW_LOCAL_POINTS_BY_STYLE[cfg.global.arrowStyle];

      function projectPoint(p, mat){
        var wp = p.clone().applyMatrix4(mat);
        var proj = wp.clone().project(camera);
        return [(proj.x*0.5+0.5)*w, (1-(proj.y*0.5+0.5))*h];
      }
      function polyTag(pts2d){
        var s = pts2d.map(function(p){ return p[0].toFixed(1)+','+p[1].toFixed(1); }).join(' ');
        return '<polygon points="'+s+'"/>';
      }

      // GLOBE mode: on screen, the opaque sphere mesh depth-occludes arrows on
      // its far side, so only the near hemisphere is ever visible. SVG has no
      // depth buffer, so every particle would otherwise get drawn -- dumping
      // the whole back side of the globe on top of the front. Cull to the
      // camera-facing hemisphere here, with a small margin that also drops
      // the near-grazing arrows right at the silhouette edge (exactly the
      // ones most prone to extreme foreshortening artifacts below).
      var globeCenter = null, globeToCam = null;
      if(state.mode === 'globe'){
        globeCenter = globeSphereCenter(new THREE.Vector3());
        if(flatMode) globeCenter.z *= FLAT_Z_SQUASH;
        globeToCam = camera.position.clone().sub(globeCenter).normalize();
      }
      var VISIBLE_MARGIN = 0.08;

      for(var i=0;i<frameArrows.length;i++){
        var a = frameArrows[i];
        if(globeCenter){
          var surfN = a.pos.clone().sub(globeCenter).normalize();
          if(surfN.dot(globeToCam) < VISIBLE_MARGIN) continue;
        }
        var widthScale = isFlat ? a.len : coneWidthScale;
        if(isFlat){ computeFlatOrientation(a.pos, a.dir, camera.position, tmpQuat); }
        else { tmpQuat.setFromUnitVectors(X_AXIS, a.dir); }
        tmpScale.set(a.len, widthScale, widthScale);
        tmpMat.compose(a.pos, tmpQuat, tmpScale);

        var polys = [];
        if(cfg.global.arrowStyle === 'cone'){
          // Solid of revolution, so its silhouette's SHAPE (the concave
          // shoulder profile, CONE_SILHOUETTE_POINTS) is the same from any
          // angle -- but its APPARENT LENGTH must foreshorten correctly as
          // the view becomes more end-on, which only happens with the arrow's
          // true 3D orientation projected through the camera's real
          // perspective (tmpMat, already composed above, exactly matching
          // the live mesh). Billboarding it toward the camera (as tried
          // before) throws that foreshortening away, which is exactly what
          // produced oversized, flipped-looking arrows at the sphere's limb.
          var pts2d=[];
          for(var k=0;k<CONE_SILHOUETTE_POINTS.length;k++) pts2d.push(projectPoint(CONE_SILHOUETTE_POINTS[k], tmpMat));
          polys.push(polyTag(pts2d));
        } else if(cfg.global.arrowStyle === 'flat'){
          // localPoints is the 7-point ordered CONCAVE outline (see
          // buildFlatArrowGeometry) -- project it and emit it directly, in
          // order. Taking its convex hull (as before) fills in the notch
          // between the shaft and the head and turns the arrow into a solid
          // blob/hexagon instead of an arrow.
          var pts2d=[];
          for(var k=0;k<localPoints.length;k++) pts2d.push(projectPoint(localPoints[k], tmpMat));
          polys.push(polyTag(pts2d));
        } else {
          // chevron: three separate ordered quads (shaft + two barbs)
          // concatenated -- keep them as three separate polygons rather
          // than hulling them together into one shape.
          for(var q=0;q<localPoints.length;q+=4){
            var quadPts=[];
            for(var k=q;k<q+4;k++) quadPts.push(projectPoint(localPoints[k], tmpMat));
            polys.push(polyTag(quadPts));
          }
        }
        if(a.accent) accentParts.push.apply(accentParts, polys); else baseParts.push.apply(baseParts, polys);
      }
      var svg = '<svg xmlns="http://www.w3.org/2000/svg" width="'+w+'" height="'+h+'" viewBox="0 0 '+w+' '+h+'">'+
        '<rect width="100%" height="100%" fill="#0b0b0c"/>'+
        '<g fill="#e7e5df">'+baseParts.join('')+'</g>'+
        '<g fill="#ff5a1f">'+accentParts.join('')+'</g>'+
        '</svg>';
      downloadBlob(svg, 'image/svg+xml', 'superplane-'+state.mode+'.svg');
    });

    // Keyboard shortcuts: P for PNG, S for SVG -- ignored while typing in a
    // panel input/select so they don't fire while e.g. editing a number field.
    document.addEventListener('keydown', function(e){
      if(e.metaKey || e.ctrlKey || e.altKey) return;
      var tag = (e.target && e.target.tagName) || '';
      if(tag === 'INPUT' || tag === 'SELECT' || tag === 'TEXTAREA') return;
      if(e.key === 'p' || e.key === 'P'){ btnPng.click(); }
      else if(e.key === 's' || e.key === 'S'){ btnSvg.click(); }
    });

    btnEmbed.addEventListener('click', function(){
      var overridesObj = {global:{}};
      overridesObj.global = {};
      GLOBAL_PARAMS.forEach(function(p){ overridesObj.global[p.key] = cfg.global[p.key]; });
      overridesObj.global.arrowStyle = cfg.global.arrowStyle;
      overridesObj[state.mode] = {};
      MODES[state.mode].params.forEach(function(p){ overridesObj[state.mode][p.key] = cfg[state.mode][p.key]; });

      var idSlug = 'sp-embed-' + state.mode;
      var mountOptions = {mode: state.mode, panel: false, overrides: overridesObj,
        camera: {theta: orbit.theta, phi: orbit.phi, radius: orbit.radius},
        cameraLocked: cameraLocked, flatMode: flatMode};
      var snippet =
        '<!-- SuperPlane generative art \u2014 fills 100% of this block\'s width/height.\n' +
        '     Set the width/height on the wrapping element on your page. -->\n' +
        '<script src="https://cdnjs.cloudflare.com/ajax/libs/three.js/r128/three.min.js"><\/script>\n' +
        '<script src="https://cdn.jsdelivr.net/gh/metabrand-agency/superplane@main/superplane-art.js"><\/script>\n' +
        '<div id="' + idSlug + '" style="width:100%;height:100%;"></div>\n' +
        '<script>SuperplaneArt.mount(\'#' + idSlug + '\', ' + JSON.stringify(mountOptions) + ');<\/script>';

      showCodeModal('EMBED CODE \u2014 paste into any page (' + state.mode.toUpperCase() + ', current settings, no panel, fully interactive)', snippet);
    });
  }

  /* ---------------- main loop ---------------- */
  var clock = new THREE.Clock();
  var elapsed = 0;
  var rafId = null;
  var destroyed = false;

  function animate(){
    if(destroyed) return;
    rafId = requestAnimationFrame(animate);
    var dt = Math.min(0.05, clock.getDelta());
    elapsed += dt;

    if(state.mode==='field') updateField(dt, elapsed);
    else if(state.mode==='school') updateSchool(dt);
    else if(state.mode==='growth') updateGrowth(dt, elapsed);
    else if(state.mode==='network') updateNetwork(dt, elapsed);
    else if(state.mode==='globe') updateGlobe(dt, elapsed);
    else if(state.mode==='startrek') updateStarTrek(dt);
    else if(state.mode==='spiral') updateSpiral(dt, elapsed);

    composeForMode(elapsed);

    if(state.autoRotate && !cameraLocked) orbit.theta += dt*0.15;
    applyOrbit();

    renderer.render(scene, camera);

    if(showPanel && readoutEl){
      var activeCount = state.mode==='field' ? fieldParticles.length :
                         state.mode==='school' ? agents.length :
                         state.mode==='network' ? networkNodes.length :
                         state.mode==='globe' ? globeParticles.length :
                         state.mode==='startrek' ? starParticles.length :
                         state.mode==='spiral' ? spiralParticles.length : segments.length;
      readoutEl.innerHTML = 'MODE &nbsp;: <b>'+state.mode.toUpperCase()+'</b><br>ARROWS: <b>'+activeCount+'</b><br>TIME &nbsp;: <b>'+elapsed.toFixed(1)+'s</b>';
    }
    if(camCoordsEl){
      var thetaDeg = ((orbit.theta*180/Math.PI) % 360 + 360) % 360;
      var phiDeg = orbit.phi*180/Math.PI;
      camCoordsEl.innerHTML = 'THETA &nbsp;: '+thetaDeg.toFixed(1)+'&deg;<br>PHI &nbsp;&nbsp;&nbsp;: '+phiDeg.toFixed(1)+'&deg;<br>RADIUS: '+orbit.radius.toFixed(2);
    }
  }
  animate();

  // Live colour tween: smoothly interpolates background/fog/sphere and arrow
  // colour on this ALREADY-RUNNING instance -- the particle system itself is
  // untouched (never re-initialised), so arrows keep animating the whole
  // time instead of disappearing the way a full mount()-over-mount() swap
  // (or a crossfade between two separate live instances) would cause.
  var colorTweenRaf = null;
  // cache-bust marker v2
  function setColors(targetBg, targetArrow, durationMs, onDone){
    if(colorTweenRaf) cancelAnimationFrame(colorTweenRaf);
    var fromBg = renderer.getClearColor(new THREE.Color());
    var toBg = new THREE.Color(targetBg);
    var fromArrow = instanceBaseMat.color.clone();
    var toArrow = new THREE.Color(targetArrow);
    var sameMat = (instanceAccentMat === instanceBaseMat);
    var fromAccent = sameMat ? null : instanceAccentMat.color.clone();
    var start = null;
    var tmp = new THREE.Color();
    function step(ts){
      if(destroyed) return;
      if(start === null) start = ts;
      var t = Math.min(1, (ts - start) / durationMs);
      // ease-in-out, matches the cubic-bezier(0.4,0,0.2,1) feel used elsewhere on the site
      var e = t < 0.5 ? 2*t*t : 1 - Math.pow(-2*t + 2, 2) / 2;
      tmp.lerpColors(fromBg, toBg, e);
      renderer.setClearColor(tmp, 1);
      if(scene.fog) scene.fog.color.lerpColors(fromBg, toBg, e);
      if(options.minimalSphere) globeSolidMat.color.lerpColors(fromBg, toBg, e);
      instanceBaseMat.color.lerpColors(fromArrow, toArrow, e);
      if(sameMat){ /* same material object, already updated above */ }
      else if(fromAccent){ instanceAccentMat.color.lerpColors(fromAccent, toArrow, e); }
      if(t < 1){
        colorTweenRaf = requestAnimationFrame(step);
      } else {
        colorTweenRaf = null;
        effectiveBgColor = targetBg;
        if(onDone) onDone();
      }
    }
    colorTweenRaf = requestAnimationFrame(step);
  }

  return {
    setColors: setColors,
    destroy: function(){
      destroyed = true;
      if(colorTweenRaf) cancelAnimationFrame(colorTweenRaf);
      if(rafId) cancelAnimationFrame(rafId);
      resizeObs.disconnect();
      renderer.dispose();
      container.innerHTML = '';
    }
  };
}

global.SuperplaneArt = { mount: mount, MODES: MODES };

})(window);
