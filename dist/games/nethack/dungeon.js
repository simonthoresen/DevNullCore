// dungeon.js — Procedural dungeon generation for nethack
// Generates rooms connected by corridors, places doors, stairs, items, and monsters.

// ─── RNG ───────────────────────────────────────────────────────────────────
function randInt(min, max) {
    return Math.floor(Math.random() * (max - min + 1)) + min;
}

function randPick(arr) {
    return arr[Math.floor(Math.random() * arr.length)];
}

// ─── Level Generation ──────────────────────────────────────────────────────

function generateLevel(width, height, depth) {
    // Create empty grid
    var grid = [];
    for (var y = 0; y < height; y++) {
        var row = [];
        for (var x = 0; x < width; x++) {
            row.push(TILES.VOID);
        }
        grid.push(row);
    }

    // Generate rooms
    var rooms = [];
    var maxRooms = 6 + Math.min(depth, 4);
    var attempts = 0;
    while (rooms.length < maxRooms && attempts < 200) {
        attempts++;
        var rw = randInt(4, 10);
        var rh = randInt(3, 7);
        var rx = randInt(1, width - rw - 2);
        var ry = randInt(1, height - rh - 2);

        // Check for overlap with existing rooms (with 1-tile padding)
        var overlaps = false;
        for (var i = 0; i < rooms.length; i++) {
            var r = rooms[i];
            if (rx - 1 < r.x + r.w + 1 && rx + rw + 1 > r.x - 1 &&
                ry - 1 < r.y + r.h + 1 && ry + rh + 1 > r.y - 1) {
                overlaps = true;
                break;
            }
        }
        if (overlaps) continue;

        // Carve room
        for (var y = ry; y < ry + rh; y++) {
            for (var x = rx; x < rx + rw; x++) {
                grid[y][x] = TILES.FLOOR;
            }
        }

        // Add walls around room
        for (var y = ry - 1; y <= ry + rh; y++) {
            for (var x = rx - 1; x <= rx + rw; x++) {
                if (y >= 0 && y < height && x >= 0 && x < width) {
                    if (grid[y][x] === TILES.VOID) {
                        grid[y][x] = TILES.WALL;
                    }
                }
            }
        }

        rooms.push({ x: rx, y: ry, w: rw, h: rh,
            cx: Math.floor(rx + rw / 2),
            cy: Math.floor(ry + rh / 2)
        });
    }

    // Connect rooms with corridors
    for (var i = 1; i < rooms.length; i++) {
        var a = rooms[i - 1];
        var b = rooms[i];
        carveCorridor(grid, a.cx, a.cy, b.cx, b.cy, width, height);
    }
    // Extra corridor for connectivity
    if (rooms.length > 2) {
        var a = rooms[0];
        var b = rooms[rooms.length - 1];
        carveCorridor(grid, a.cx, a.cy, b.cx, b.cy, width, height);
    }

    // Place stairs
    var stairsUpRoom = rooms[0];
    var stairsDownRoom = rooms[rooms.length - 1];
    var stairsUp = { x: stairsUpRoom.cx, y: stairsUpRoom.cy };
    var stairsDown = { x: stairsDownRoom.cx, y: stairsDownRoom.cy };
    grid[stairsUp.y][stairsUp.x] = TILES.STAIRS_UP;
    grid[stairsDown.y][stairsDown.x] = TILES.STAIRS_DOWN;

    // Place items
    var items = [];
    var numItems = randInt(3, 5 + depth);
    var availableItems = getItemsForDepth(depth);
    for (var i = 0; i < numItems && availableItems.length > 0; i++) {
        var room = randPick(rooms);
        var ix = randInt(room.x, room.x + room.w - 1);
        var iy = randInt(room.y, room.y + room.h - 1);
        if (grid[iy][ix] === TILES.FLOOR) {
            var itemDef = randPick(availableItems);
            items.push({
                x: ix, y: iy,
                category: itemDef.category,
                def: itemDef.def,
                id: 'item_' + depth + '_' + i
            });
        }
    }

    // Place gold piles
    var numGold = randInt(1, 3 + depth);
    for (var i = 0; i < numGold; i++) {
        var room = randPick(rooms);
        var gx = randInt(room.x, room.x + room.w - 1);
        var gy = randInt(room.y, room.y + room.h - 1);
        if (grid[gy][gx] === TILES.FLOOR) {
            items.push({
                x: gx, y: gy,
                category: 'gold',
                def: { ch: '$', name: 'gold', value: randInt(5, 15) * depth },
                id: 'gold_' + depth + '_' + i
            });
        }
    }

    // Place traps (from depth 2+)
    var traps = [];
    if (depth >= 2) {
        var numTraps = randInt(0, Math.min(depth - 1, 3));
        for (var i = 0; i < numTraps; i++) {
            var room = randPick(rooms);
            var tx = randInt(room.x, room.x + room.w - 1);
            var ty = randInt(room.y, room.y + room.h - 1);
            if (grid[ty][tx] === TILES.FLOOR) {
                traps.push({
                    x: tx, y: ty,
                    def: randPick(TRAP_DEFS),
                    revealed: false
                });
            }
        }
    }

    // Place monsters
    var monsters = [];
    var numMonsters = randInt(3, 4 + depth);
    var availableMonsters = getMonstersForDepth(depth);
    for (var i = 0; i < numMonsters && availableMonsters.length > 0; i++) {
        var room = randPick(rooms);
        var mx = randInt(room.x, room.x + room.w - 1);
        var my = randInt(room.y, room.y + room.h - 1);
        if (grid[my][mx] === TILES.FLOOR &&
            !(mx === stairsUp.x && my === stairsUp.y)) {
            var def = randPick(availableMonsters);
            monsters.push(createMonster(def, mx, my, depth + '_' + i));
        }
    }

    return {
        grid: grid,
        width: width,
        height: height,
        depth: depth,
        rooms: rooms,
        stairsUp: stairsUp,
        stairsDown: stairsDown,
        items: items,
        traps: traps,
        monsters: monsters
    };
}

function carveCorridor(grid, x1, y1, x2, y2, width, height) {
    var x = x1;
    var y = y1;

    // Go horizontal first, then vertical (or vice versa randomly)
    if (Math.random() < 0.5) {
        while (x !== x2) {
            x += (x2 > x) ? 1 : -1;
            carvePoint(grid, x, y, width, height);
        }
        while (y !== y2) {
            y += (y2 > y) ? 1 : -1;
            carvePoint(grid, x, y, width, height);
        }
    } else {
        while (y !== y2) {
            y += (y2 > y) ? 1 : -1;
            carvePoint(grid, x, y, width, height);
        }
        while (x !== x2) {
            x += (x2 > x) ? 1 : -1;
            carvePoint(grid, x, y, width, height);
        }
    }
}

function carvePoint(grid, x, y, width, height) {
    if (x < 0 || x >= width || y < 0 || y >= height) return;
    if (grid[y][x] === TILES.VOID) {
        grid[y][x] = TILES.CORRIDOR;
        // Add walls around corridor
        for (var dy = -1; dy <= 1; dy++) {
            for (var dx = -1; dx <= 1; dx++) {
                var ny = y + dy;
                var nx = x + dx;
                if (ny >= 0 && ny < height && nx >= 0 && nx < width) {
                    if (grid[ny][nx] === TILES.VOID) {
                        grid[ny][nx] = TILES.WALL;
                    }
                }
            }
        }
    }
}

// Check if a tile is walkable
function isWalkable(tile) {
    return tile === TILES.FLOOR || tile === TILES.CORRIDOR ||
           tile === TILES.DOOR_OPEN || tile === TILES.STAIRS_DOWN ||
           tile === TILES.STAIRS_UP;
}

// Find a random floor tile in a room for spawning
function findSpawnPoint(level) {
    for (var attempts = 0; attempts < 100; attempts++) {
        var room = randPick(level.rooms);
        var x = randInt(room.x, room.x + room.w - 1);
        var y = randInt(room.y, room.y + room.h - 1);
        if (isWalkable(level.grid[y][x])) {
            // Check no monster or player here
            var occupied = false;
            for (var m = 0; m < level.monsters.length; m++) {
                if (level.monsters[m].x === x && level.monsters[m].y === y && level.monsters[m].hp > 0) {
                    occupied = true;
                    break;
                }
            }
            if (!occupied) return { x: x, y: y };
        }
    }
    // Fallback: stairs up position
    return { x: level.stairsUp.x, y: level.stairsUp.y };
}
