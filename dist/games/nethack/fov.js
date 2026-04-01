// fov.js — Field-of-view using recursive shadowcasting
// Each player has their own visibility map. Explored tiles remain dimly visible.

var FOV_RADIUS = 8;

// Compute FOV for a given position on a level grid.
// Returns a 2D boolean array [y][x] = true if visible.
function computeFOV(grid, px, py, width, height) {
    var visible = [];
    for (var y = 0; y < height; y++) {
        var row = [];
        for (var x = 0; x < width; x++) {
            row.push(false);
        }
        visible.push(row);
    }

    // The origin is always visible
    visible[py][px] = true;

    // Cast in all 8 octants
    for (var octant = 0; octant < 8; octant++) {
        castLight(grid, visible, px, py, width, height, 1, 1.0, 0.0, octant);
    }

    return visible;
}

function castLight(grid, visible, cx, cy, width, height, row, startSlope, endSlope, octant) {
    if (startSlope < endSlope) return;

    var nextStart = startSlope;

    for (var i = row; i <= FOV_RADIUS; i++) {
        var blocked = false;

        for (var dx = -i; dx <= 0; dx++) {
            var dy = -i;

            // Transform based on octant
            var tx, ty;
            switch (octant) {
                case 0: tx = cx + dx; ty = cy + dy; break;
                case 1: tx = cx + dy; ty = cy + dx; break;
                case 2: tx = cx + dy; ty = cy - dx; break;
                case 3: tx = cx + dx; ty = cy - dy; break;
                case 4: tx = cx - dx; ty = cy - dy; break;
                case 5: tx = cx - dy; ty = cy - dx; break;
                case 6: tx = cx - dy; ty = cy + dx; break;
                case 7: tx = cx - dx; ty = cy + dy; break;
            }

            if (tx < 0 || tx >= width || ty < 0 || ty >= height) continue;

            var lSlope = (dx - 0.5) / (dy + 0.5);
            var rSlope = (dx + 0.5) / (dy - 0.5);

            if (startSlope < rSlope) continue;
            if (endSlope > lSlope) break;

            // Distance check (circular FOV)
            var dist = Math.sqrt(dx * dx + dy * dy);
            if (dist <= FOV_RADIUS) {
                visible[ty][tx] = true;
            }

            var isBlocking = (grid[ty][tx] === TILES.WALL);

            if (blocked) {
                if (isBlocking) {
                    nextStart = rSlope;
                } else {
                    blocked = false;
                    startSlope = nextStart;
                }
            } else if (isBlocking && i < FOV_RADIUS) {
                blocked = true;
                castLight(grid, visible, cx, cy, width, height, i + 1, startSlope, lSlope, octant);
                nextStart = rSlope;
            }
        }
        if (blocked) break;
    }
}

// Update a player's explored map based on current visibility
function updateExplored(explored, visible, height, width) {
    for (var y = 0; y < height; y++) {
        for (var x = 0; x < width; x++) {
            if (visible[y][x]) {
                explored[y][x] = true;
            }
        }
    }
}

// Create an empty explored map
function createExploredMap(width, height) {
    var explored = [];
    for (var y = 0; y < height; y++) {
        var row = [];
        for (var x = 0; x < width; x++) {
            row.push(false);
        }
        explored.push(row);
    }
    return explored;
}
