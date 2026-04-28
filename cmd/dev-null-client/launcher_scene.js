// Launcher background scene.
//
// Renders a slowly-rotating, lit 3D cube on a starfield using the engine's
// triangle rasterizer (canvas.fillTriangle3DLit). The same canvas + raster
// pipeline drives in-game 3D rendering, so the launcher exercises exactly
// the same code path that ships to players.

var Game = {
  resolveMe: function(state, pid) { return { id: pid }; },
  renderCanvas: function(state, me, canvas) {
    var w = canvas.width, h = canvas.height;
    var t = state._gameTime || 0;

    function clampByte(v) {
      if (v < 0) return 0;
      if (v > 255) return 255;
      return v | 0;
    }
    function hex2(v) {
      var s = clampByte(v).toString(16);
      return s.length < 2 ? "0" + s : s;
    }
    function rgbHex(r, g, b) {
      return "#" + hex2(r) + hex2(g) + hex2(b);
    }

    // ── 2D background ────────────────────────────────────────────────
    canvas.setFillStyle("#03050d");
    canvas.fillRect(0, 0, w, h);

    // Twinkling starfield (deterministic positions, time-modulated brightness).
    var stars = 220;
    for (var i = 0; i < stars; i++) {
      var sx = (i * 137 + 53) % w;
      var sy = (i * 89 + 17) % h;
      var tw = 0.5 + 0.5 * Math.sin(t * 1.6 + i * 0.37);
      var v = Math.floor(110 + tw * 130);
      canvas.setFillStyle(rgbHex(v, v, v + 20));
      canvas.fillRect(sx, sy, 1, 1);
    }

    // ── 3D rotating cube ─────────────────────────────────────────────
    canvas.clearDepth();

    var cx = w * 0.5, cy = h * 0.5;
    // Match the cube test's projection scaling to keep the cube visually
    // square on a 2:1 aspect terminal cell when the result is quadrantized.
    var base = Math.min(w, h) * 0.18;
    var sx = base * 2;
    var sy = base * 2;

    var ax = t * 0.55;
    var ay = t * 0.85;
    var az = t * 0.20;

    var cosX = Math.cos(ax), sinX = Math.sin(ax);
    var cosY = Math.cos(ay), sinY = Math.sin(ay);
    var cosZ = Math.cos(az), sinZ = Math.sin(az);

    function rotate(v) {
      var x = v[0], y = v[1], z = v[2];
      // Y axis
      var rx = x * cosY + z * sinY;
      var rz = -x * sinY + z * cosY;
      // X axis
      var ry = y * cosX - rz * sinX;
      rz = y * sinX + rz * cosX;
      // Z axis
      var fx = rx * cosZ - ry * sinZ;
      var fy = rx * sinZ + ry * cosZ;
      return [fx, fy, rz];
    }

    function project(v) {
      // Move cube away from camera, perspective divide.
      var z = v[2] + 4.5;
      var d = 4.0 / z;
      // Rasterizer expects screen-space pixels for x/y; smaller z is closer.
      return [cx + v[0] * sx * d, cy + v[1] * sy * d, z];
    }

    var verts = [
      [-1, -1, -1], [ 1, -1, -1], [ 1,  1, -1], [-1,  1, -1],
      [-1, -1,  1], [ 1, -1,  1], [ 1,  1,  1], [-1,  1,  1],
    ];
    var pv = [];
    var rv = [];
    for (var i = 0; i < verts.length; i++) {
      var r = rotate(verts[i]);
      rv.push(r);
      pv.push(project(r));
    }

    // Each face: 4 vertex indices + base color. Two triangles per face.
    var faces = [
      { idx: [0, 1, 2, 3], color: "#5c6bc0" }, // -Z
      { idx: [5, 4, 7, 6], color: "#42a5f5" }, // +Z
      { idx: [4, 0, 3, 7], color: "#26a69a" }, // -X
      { idx: [1, 5, 6, 2], color: "#ef6c00" }, // +X
      { idx: [4, 5, 1, 0], color: "#7e57c2" }, // -Y
      { idx: [3, 2, 6, 7], color: "#ec407a" }, // +Y
    ];

    var lightDir = [-0.4, -0.6, -0.7];

    for (var f = 0; f < faces.length; f++) {
      var face = faces[f];
      var a = rv[face.idx[0]];
      var b = rv[face.idx[1]];
      var c = rv[face.idx[2]];
      // Face normal in rotated (camera) space.
      var ux = b[0] - a[0], uy = b[1] - a[1], uz = b[2] - a[2];
      var vx = c[0] - a[0], vy = c[1] - a[1], vz = c[2] - a[2];
      var nx = uy * vz - uz * vy;
      var ny = uz * vx - ux * vz;
      var nz = ux * vy - uy * vx;
      var nlen = Math.sqrt(nx * nx + ny * ny + nz * nz) || 1;
      nx /= nlen; ny /= nlen; nz /= nlen;
      var n = [nx, ny, nz];

      var p0 = pv[face.idx[0]];
      var p1 = pv[face.idx[1]];
      var p2 = pv[face.idx[2]];
      var p3 = pv[face.idx[3]];

      canvas.fillTriangle3DLit(p0, p1, p2, n, n, n, lightDir, face.color, 0.18);
      canvas.fillTriangle3DLit(p0, p2, p3, n, n, n, lightDir, face.color, 0.18);
    }
  }
};
