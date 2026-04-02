// ui.js — Rendering functions for nethack

// ─── Main View Renderer ───────────────────────────────────────────────────

function renderView(buf, player, level, players, width, height) {
    if (player.dead) {
        renderDeathScreen(buf, player, width, height);
        return;
    }

    // Compute FOV from player position
    var visible = computeFOV(level.grid, player.x, player.y, level.width, level.height);
    updateExplored(player.explored, visible, level.height, level.width);

    // Determine viewport (centered on player)
    var vpw = width;
    var vph = height - 3; // reserve 3 lines for messages at bottom
    if (vph < 5) vph = height; // if too small, use full height

    var camX = player.x - Math.floor(vpw / 2);
    var camY = player.y - Math.floor(vph / 2);
    // Clamp camera
    if (camX < 0) camX = 0;
    if (camY < 0) camY = 0;
    if (camX + vpw > level.width) camX = Math.max(0, level.width - vpw);
    if (camY + vph > level.height) camY = Math.max(0, level.height - vph);

    for (var vy = 0; vy < vph; vy++) {
        var gy = camY + vy;
        for (var vx = 0; vx < vpw; vx++) {
            var gx = camX + vx;

            if (gx < 0 || gx >= level.width || gy < 0 || gy >= level.height) {
                continue; // buffer is pre-filled with spaces
            }

            var isVisible = visible[gy][gx];
            var isExplored = player.explored[gy][gx];

            if (!isVisible && !isExplored) {
                continue; // buffer is pre-filled with spaces
            }

            // Check for entities (only if visible)
            var ch = null;
            if (isVisible) {
                // Check for players
                for (var pid in players) {
                    var p = players[pid];
                    if (!p.dead && p.depth === player.depth && p.x === gx && p.y === gy) {
                        ch = (pid === player.id) ? '@' : '&';
                        break;
                    }
                }
                // Check for monsters
                if (!ch) {
                    for (var m = 0; m < level.monsters.length; m++) {
                        var mon = level.monsters[m];
                        if (mon.hp > 0 && mon.x === gx && mon.y === gy) {
                            ch = mon.ch;
                            break;
                        }
                    }
                }
                // Check for items (show topmost)
                if (!ch) {
                    for (var i = level.items.length - 1; i >= 0; i--) {
                        if (level.items[i].x === gx && level.items[i].y === gy) {
                            ch = level.items[i].def.ch;
                            break;
                        }
                    }
                }
                // Check for revealed traps
                if (!ch) {
                    for (var t = 0; t < level.traps.length; t++) {
                        if (level.traps[t].revealed && level.traps[t].x === gx && level.traps[t].y === gy) {
                            ch = TILES.TRAP;
                            break;
                        }
                    }
                }
            }

            if (!ch) {
                ch = level.grid[gy][gx];
                // Dim explored-but-not-visible tiles
                if (!isVisible && isExplored) {
                    if (ch === TILES.FLOOR || ch === TILES.CORRIDOR) ch = ':';
                    else if (ch === TILES.WALL) ch = '#';
                }
            }

            buf.setChar(vx, vy, ch, null, null);
        }
    }

    // Add message log at bottom
    var msgLines = Math.min(3, height - vph);
    if (msgLines > 0) {
        var msgs = player.messages.slice(-msgLines);
        while (msgs.length < msgLines) {
            msgs.unshift('');
        }
        for (var i = 0; i < msgs.length; i++) {
            var msg = msgs[i];
            if (msg.length > width) msg = msg.substring(0, width);
            buf.writeString(0, vph + i, msg, null, null);
        }
    }
}

// ─── Death Screen ──────────────────────────────────────────────────────────

function renderDeathScreen(buf, player, width, height) {
    var center = Math.floor(height / 2);
    var texts = [
        '--- REST IN PEACE ---',
        '',
        player.name,
        '',
        'Level ' + player.level + ' adventurer',
        'Killed on depth ' + player.depth,
        'with ' + player.gold + ' gold',
        player.kills + ' monsters slain',
        '',
        'Press [r] to respawn'
    ];

    for (var i = 0; i < texts.length; i++) {
        var row = center - Math.floor(texts.length / 2) + i;
        if (row >= 0 && row < height && texts[i].length > 0) {
            var pad = Math.floor((width - texts[i].length) / 2);
            if (pad < 0) pad = 0;
            buf.writeString(pad, row, texts[i], null, null);
        }
    }
}

// ─── Inventory View ────────────────────────────────────────────────────────

function renderInventory(buf, player, width, height) {
    var items = [];
    items.push('--- Inventory ---');
    items.push('');
    if (player.weapon) {
        items.push('Weapon: ' + player.weapon.name + ' (+' + player.weapon.atk + ' atk)');
    } else {
        items.push('Weapon: (none)');
    }
    if (player.armor) {
        items.push('Armor:  ' + player.armor.name + ' (+' + player.armor.def + ' def)');
    } else {
        items.push('Armor:  (none)');
    }
    items.push('');

    for (var i = 0; i < player.inventory.length; i++) {
        var it = player.inventory[i];
        var letter = String.fromCharCode(97 + i); // a, b, c...
        var desc = letter + ') ' + it.def.ch + ' ' + it.def.name;
        if (it.category === 'weapons') desc += ' (+' + it.def.atk + ' atk)';
        if (it.category === 'armor') desc += ' (+' + it.def.def + ' def)';
        items.push(desc);
    }
    if (player.inventory.length === 0) {
        items.push('(empty)');
    }
    items.push('');
    items.push('[a-o] Use item  [Esc] Close');

    var startRow = Math.max(0, Math.floor((height - items.length) / 2));
    for (var i = 0; i < items.length; i++) {
        var row = startRow + i;
        if (row >= height) break;
        if (items[i].length === 0) continue;
        var pad = Math.floor((width - items[i].length) / 2);
        if (pad < 0) pad = 0;
        buf.writeString(pad, row, items[i], null, null);
    }
}

// ─── Status Bar ────────────────────────────────────────────────────────────

function renderStatusBar(player) {
    if (player.dead) {
        return 'DEAD  |  ' + player.name + '  |  Gold: ' + player.gold + '  |  Kills: ' + player.kills;
    }
    var wpn = player.weapon ? player.weapon.name : 'fists';
    var arm = player.armor ? player.armor.name : 'none';
    var hungerStr = '';
    if (player.hunger < 50) hungerStr = '  STARVING!';
    else if (player.hunger < 150) hungerStr = '  Hungry';

    return 'HP:' + player.hp + '/' + player.maxHp +
           '  Lv:' + player.level +
           '  Dp:' + player.depth +
           '  Atk:' + (player.atk + (player.weapon ? player.weapon.atk : 0)) +
           '  Def:' + (player.def + (player.armor ? player.armor.def : 0)) +
           '  XP:' + player.xp + '/' + player.xpToLevel +
           '  $' + player.gold +
           hungerStr;
}

// ─── Command Bar ───────────────────────────────────────────────────────────

function renderCommandBar(player, showInventory) {
    if (player.dead) {
        return '[r] Respawn  [Enter] Chat';
    }
    if (showInventory) {
        return '[a-o] Use item  [Esc] Close inventory';
    }
    return '[arrows] Move  [g] Grab  [i] Inventory  [>] Descend  [<] Ascend  [.] Wait  [Enter] Chat';
}
