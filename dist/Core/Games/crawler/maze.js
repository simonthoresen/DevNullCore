// maze.js — Maze generation using recursive backtracker

var TILE_WALL = 1;
var TILE_FLOOR = 0;
var TILE_STAIRS = 2;

function generateMaze(w, h) {
    // Initialize all walls
    var grid = [];
    for (var y = 0; y < h; y++) {
        var row = [];
        for (var x = 0; x < w; x++) {
            row.push(TILE_WALL);
        }
        grid.push(row);
    }

    // Recursive backtracker on odd cells
    var visited = {};
    var stack = [];
    var sx = 1, sy = 1;
    grid[sy][sx] = TILE_FLOOR;
    visited[sy * w + sx] = true;
    stack.push({ x: sx, y: sy });

    while (stack.length > 0) {
        var cur = stack[stack.length - 1];
        var neighbors = [];
        var dirs = [
            { dx: 0, dy: -2 },
            { dx: 0, dy: 2 },
            { dx: -2, dy: 0 },
            { dx: 2, dy: 0 }
        ];

        for (var d = 0; d < dirs.length; d++) {
            var nx = cur.x + dirs[d].dx;
            var ny = cur.y + dirs[d].dy;
            if (nx > 0 && nx < w - 1 && ny > 0 && ny < h - 1 && !visited[ny * w + nx]) {
                neighbors.push({ x: nx, y: ny, wx: cur.x + dirs[d].dx / 2, wy: cur.y + dirs[d].dy / 2 });
            }
        }

        if (neighbors.length > 0) {
            var next = neighbors[Math.floor(Math.random() * neighbors.length)];
            grid[next.wy][next.wx] = TILE_FLOOR;
            grid[next.y][next.x] = TILE_FLOOR;
            visited[next.y * w + next.x] = true;
            stack.push({ x: next.x, y: next.y });
        } else {
            stack.pop();
        }
    }

    // Add some extra passages for less linearity
    var extraPasses = Math.floor(w * h * 0.03);
    for (var i = 0; i < extraPasses; i++) {
        var rx = 1 + Math.floor(Math.random() * (w - 2));
        var ry = 1 + Math.floor(Math.random() * (h - 2));
        if (grid[ry][rx] === TILE_WALL) {
            // Check if removing this wall connects two floor tiles
            var adjFloors = 0;
            if (ry > 0 && grid[ry - 1][rx] === TILE_FLOOR) adjFloors++;
            if (ry < h - 1 && grid[ry + 1][rx] === TILE_FLOOR) adjFloors++;
            if (rx > 0 && grid[ry][rx - 1] === TILE_FLOOR) adjFloors++;
            if (rx < w - 1 && grid[ry][rx + 1] === TILE_FLOOR) adjFloors++;
            if (adjFloors >= 2) {
                grid[ry][rx] = TILE_FLOOR;
            }
        }
    }

    // Place stairs at a far point from start
    var stairsX = -1, stairsY = -1;
    var maxDist = 0;
    for (var y = 1; y < h - 1; y++) {
        for (var x = 1; x < w - 1; x++) {
            if (grid[y][x] === TILE_FLOOR) {
                var dist = Math.abs(x - sx) + Math.abs(y - sy);
                if (dist > maxDist) {
                    maxDist = dist;
                    stairsX = x;
                    stairsY = y;
                }
            }
        }
    }
    if (stairsX >= 0) {
        grid[stairsY][stairsX] = TILE_STAIRS;
    }

    return { grid: grid, w: w, h: h, startX: sx, startY: sy, stairsX: stairsX, stairsY: stairsY };
}

function findSpawn(maze) {
    return { x: maze.startX, y: maze.startY };
}

function canWalk(maze, x, y, margin) {
    // Check all four corners of the player's bounding box
    var checks = [
        { x: x - margin, y: y - margin },
        { x: x + margin, y: y - margin },
        { x: x - margin, y: y + margin },
        { x: x + margin, y: y + margin }
    ];
    for (var i = 0; i < checks.length; i++) {
        var cx = Math.floor(checks[i].x);
        var cy = Math.floor(checks[i].y);
        if (cx < 0 || cx >= maze.w || cy < 0 || cy >= maze.h) return false;
        if (maze.grid[cy][cx] === TILE_WALL) return false;
    }
    return true;
}
