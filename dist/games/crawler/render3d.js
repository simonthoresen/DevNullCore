// render3d.js — First-person ASCII raycasting renderer

var FOV = Math.PI / 3;       // 60 degree field of view
var MAX_DEPTH = 20;
var RAY_STEP = 0.05;

// Wall shading characters by distance
var SHADE_CHARS = ['\u2588', '\u2593', '\u2592', '\u2591', '.'];
// █ ▓ ▒ ░ .

// Wall colors by distance (near to far)
var WALL_COLORS_NS = ['#ffffff', '#cccccc', '#999999', '#666666', '#444444'];
var WALL_COLORS_EW = ['#dddddd', '#aaaaaa', '#888888', '#555555', '#333333'];

// Floor/ceiling shading
var FLOOR_CHARS = ['_', '-', '.', ' '];
var CEIL_CHARS = ['^', '~', '`', ' '];

function render3D(buf, player, maze, monsters, items, ox, oy, w, h) {
    var halfH = Math.floor(h / 2);

    // Cast a ray for each column
    for (var col = 0; col < w; col++) {
        var rayAngle = player.angle - FOV / 2 + (col / w) * FOV;
        var cosA = Math.cos(rayAngle);
        var sinA = Math.sin(rayAngle);

        // March the ray
        var dist = 0;
        var hitWall = false;
        var hitSide = 0; // 0 = NS wall, 1 = EW wall
        var hitX = 0, hitY = 0;

        while (dist < MAX_DEPTH && !hitWall) {
            dist += RAY_STEP;
            hitX = player.x + cosA * dist;
            hitY = player.y + sinA * dist;

            var cellX = Math.floor(hitX);
            var cellY = Math.floor(hitY);

            if (cellX < 0 || cellX >= maze.w || cellY < 0 || cellY >= maze.h) {
                hitWall = true;
                dist = MAX_DEPTH;
                break;
            }

            if (maze.grid[cellY][cellX] === TILE_WALL) {
                hitWall = true;

                // Determine which side was hit for shading
                var fracX = hitX - cellX;
                var fracY = hitY - cellY;
                var edgeDistX = Math.min(fracX, 1 - fracX);
                var edgeDistY = Math.min(fracY, 1 - fracY);
                hitSide = (edgeDistX < edgeDistY) ? 0 : 1;
            }
        }

        // Fix fisheye by multiplying with cos of angle offset
        var angleDiff = rayAngle - player.angle;
        var correctedDist = dist * Math.cos(angleDiff);
        if (correctedDist < 0.1) correctedDist = 0.1;

        // Calculate wall height
        var wallHeight = Math.floor(h / correctedDist);
        var wallTop = halfH - Math.floor(wallHeight / 2);
        var wallBottom = halfH + Math.floor(wallHeight / 2);

        // Draw column
        for (var row = 0; row < h; row++) {
            var ch, fg, bg;

            if (row < wallTop) {
                // Ceiling
                var ceilDist = (h - row) / halfH;
                var ceilIdx = Math.min(Math.floor(ceilDist * CEIL_CHARS.length * 0.5), CEIL_CHARS.length - 1);
                ch = CEIL_CHARS[ceilIdx];
                fg = '#222244';
                bg = '#000011';
            } else if (row >= wallTop && row < wallBottom) {
                // Wall
                var shadeIdx = Math.floor(correctedDist / MAX_DEPTH * SHADE_CHARS.length);
                if (shadeIdx >= SHADE_CHARS.length) shadeIdx = SHADE_CHARS.length - 1;
                if (shadeIdx < 0) shadeIdx = 0;

                ch = SHADE_CHARS[shadeIdx];
                var colors = (hitSide === 0) ? WALL_COLORS_NS : WALL_COLORS_EW;
                fg = colors[shadeIdx];
                bg = null;

                // Highlight stairs
                if (hitWall) {
                    var cx = Math.floor(hitX);
                    var cy = Math.floor(hitY);
                    // Check adjacent cells for stairs (so player sees the doorway glow)
                    if (isNearStairs(maze, cx, cy)) {
                        fg = blendColor(fg, '#44ff44', 0.3);
                    }
                }
            } else {
                // Floor
                var floorDist = (row - halfH) / halfH;
                if (floorDist < 0.01) floorDist = 0.01;
                var floorIdx = Math.min(Math.floor((1 - floorDist) * FLOOR_CHARS.length), FLOOR_CHARS.length - 1);
                if (floorIdx < 0) floorIdx = 0;
                ch = FLOOR_CHARS[floorIdx];
                fg = '#334433';
                bg = '#001100';
            }

            buf.setChar(ox + col, oy + row, ch, fg, bg);
        }
    }

    // Render sprites (monsters and items) using painter's algorithm
    renderSprites(buf, player, monsters, items, ox, oy, w, h);

    // Draw minimap in top-left corner
    renderMinimap(buf, player, maze, monsters, items, ox, oy, w, h);

    // Compass
    renderCompass(buf, player, ox, oy, w);
}

// ─── Sprite Rendering ──────────────────────────────────────────────────────

function renderSprites(buf, player, monsters, items, ox, oy, w, h) {
    var halfH = Math.floor(h / 2);
    var sprites = [];

    // Collect visible sprites
    for (var m = 0; m < monsters.length; m++) {
        var mon = monsters[m];
        if (mon.hp <= 0) continue;
        sprites.push({ x: mon.x + 0.5, y: mon.y + 0.5, ch: mon.ch, fg: mon.color, type: 'monster', name: mon.name });
    }
    for (var i = 0; i < items.length; i++) {
        var it = items[i];
        sprites.push({ x: it.x + 0.5, y: it.y + 0.5, ch: '*', fg: '#ffff00', type: 'item', name: it.item.name });
    }

    // Sort back to front
    sprites.sort(function(a, b) {
        var da = (a.x - player.x) * (a.x - player.x) + (a.y - player.y) * (a.y - player.y);
        var db = (b.x - player.x) * (b.x - player.x) + (b.y - player.y) * (b.y - player.y);
        return db - da; // far first
    });

    for (var s = 0; s < sprites.length; s++) {
        var sp = sprites[s];
        var dx = sp.x - player.x;
        var dy = sp.y - player.y;
        var dist = Math.sqrt(dx * dx + dy * dy);
        if (dist < 0.3 || dist > MAX_DEPTH) continue;

        // Angle to sprite relative to player facing
        var spriteAngle = Math.atan2(dy, dx) - player.angle;
        // Normalize to -PI..PI
        while (spriteAngle > Math.PI) spriteAngle -= 2 * Math.PI;
        while (spriteAngle < -Math.PI) spriteAngle += 2 * Math.PI;

        // Check if in FOV
        if (Math.abs(spriteAngle) > FOV / 2 + 0.1) continue;

        // Screen position
        var screenX = Math.floor((spriteAngle / FOV + 0.5) * w);
        var spriteHeight = Math.floor(h / dist);
        var spriteTop = halfH - Math.floor(spriteHeight / 2);

        // Draw sprite character at center
        var drawY = spriteTop + Math.floor(spriteHeight * 0.3);
        if (screenX >= 0 && screenX < w && drawY >= 0 && drawY < h) {
            buf.setChar(ox + screenX, oy + drawY, sp.ch, sp.fg, null);
        }

        // Draw name above if close enough
        if (dist < 5 && sp.name) {
            var nameY = spriteTop - 1;
            if (nameY >= 0 && nameY < h) {
                var nameStart = screenX - Math.floor(sp.name.length / 2);
                for (var c = 0; c < sp.name.length; c++) {
                    var nx = nameStart + c;
                    if (nx >= 0 && nx < w) {
                        buf.setChar(ox + nx, oy + nameY, sp.name.charAt(c), sp.fg, null);
                    }
                }
            }
        }
    }
}

// ─── Minimap ───────────────────────────────────────────────────────────────

function renderMinimap(buf, player, maze, monsters, items, ox, oy, w, h) {
    var mapSize = 9; // 9x9 cells centered on player
    var mapOx = ox + 1;
    var mapOy = oy + 1;
    var half = Math.floor(mapSize / 2);
    var px = Math.floor(player.x);
    var py = Math.floor(player.y);

    for (var dy = 0; dy < mapSize; dy++) {
        for (var dx = 0; dx < mapSize; dx++) {
            var mx = px - half + dx;
            var my = py - half + dy;
            var ch = ' ';
            var fg = '#333333';
            var bg = '#111111';

            if (mx >= 0 && mx < maze.w && my >= 0 && my < maze.h) {
                var tile = maze.grid[my][mx];
                if (tile === TILE_WALL) {
                    ch = '#';
                    fg = '#555555';
                    bg = '#222222';
                } else if (tile === TILE_STAIRS) {
                    ch = '>';
                    fg = '#44ff44';
                    bg = '#002200';
                } else {
                    ch = '.';
                    fg = '#444444';
                    bg = '#111111';
                }

                // Show items
                for (var i = 0; i < items.length; i++) {
                    if (items[i].x === mx && items[i].y === my) {
                        ch = '*';
                        fg = '#ffff00';
                        break;
                    }
                }

                // Show monsters
                for (var m = 0; m < monsters.length; m++) {
                    if (monsters[m].hp > 0 && monsters[m].x === mx && monsters[m].y === my) {
                        ch = '!';
                        fg = '#ff4444';
                        break;
                    }
                }
            }

            // Player
            if (dx === half && dy === half) {
                ch = '@';
                fg = '#00ff00';
                bg = '#002200';
            }

            buf.setChar(mapOx + dx, mapOy + dy, ch, fg, bg);
        }
    }
}

// ─── Compass ───────────────────────────────────────────────────────────────

function renderCompass(buf, player, ox, oy, w) {
    var dirs = ['N', 'NE', 'E', 'SE', 'S', 'SW', 'W', 'NW'];
    // Normalize angle to 0..2PI
    var a = player.angle % (2 * Math.PI);
    if (a < 0) a += 2 * Math.PI;
    var idx = Math.round(a / (Math.PI / 4)) % 8;
    var compass = '[ ' + dirs[idx] + ' ]';
    var cx = ox + Math.floor(w / 2) - Math.floor(compass.length / 2);
    buf.writeString(cx, oy, compass, '#ffcc00', '#000000');
}

// ─── Helpers ───────────────────────────────────────────────────────────────

function isNearStairs(maze, cx, cy) {
    for (var dy = -1; dy <= 1; dy++) {
        for (var dx = -1; dx <= 1; dx++) {
            var nx = cx + dx;
            var ny = cy + dy;
            if (nx >= 0 && nx < maze.w && ny >= 0 && ny < maze.h) {
                if (maze.grid[ny][nx] === TILE_STAIRS) return true;
            }
        }
    }
    return false;
}

function blendColor(hex1, hex2, t) {
    var r1 = parseInt(hex1.substring(1, 3), 16);
    var g1 = parseInt(hex1.substring(3, 5), 16);
    var b1 = parseInt(hex1.substring(5, 7), 16);
    var r2 = parseInt(hex2.substring(1, 3), 16);
    var g2 = parseInt(hex2.substring(3, 5), 16);
    var b2 = parseInt(hex2.substring(5, 7), 16);
    var r = Math.floor(r1 + (r2 - r1) * t);
    var g = Math.floor(g1 + (g2 - g1) * t);
    var b = Math.floor(b1 + (b2 - b1) * t);
    var rh = r.toString(16); if (rh.length < 2) rh = '0' + rh;
    var gh = g.toString(16); if (gh.length < 2) gh = '0' + gh;
    var bh = b.toString(16); if (bh.length < 2) bh = '0' + bh;
    return '#' + rh + gh + bh;
}
