// entities.js — Player and monster entity logic for nethack

// ─── Player Creation ───────────────────────────────────────────────────────

function createPlayer(id, name, x, y) {
    return {
        id: id,
        name: name,
        x: x,
        y: y,
        hp: 20,
        maxHp: 20,
        atk: 3,
        def: 1,
        level: 1,
        xp: 0,
        xpToLevel: 20,
        gold: 0,
        hunger: 500,    // ticks until starving
        depth: 1,       // current dungeon depth
        weapon: null,   // equipped weapon def
        armor: null,    // equipped armor def
        inventory: [],  // array of { category, def }
        maxInventory: 15,
        dead: false,
        messages: [],   // recent messages (shown in status area)
        kills: 0,
        explored: null, // per-level explored map (reset on floor change)
        turnCount: 0
    };
}

// ─── Monster Creation ──────────────────────────────────────────────────────

function createMonster(def, x, y, id) {
    return {
        id: 'mon_' + id,
        ch: def.ch,
        name: def.name,
        x: x,
        y: y,
        hp: def.hp,
        maxHp: def.hp,
        atk: def.atk,
        def: def.def,
        speed: def.speed,
        xp: def.xp,
        hostile: true,
        lastMoveDir: null
    };
}

// ─── Combat ────────────────────────────────────────────────────────────────

function meleeAttack(attacker, defender, attackerAtk, defenderDef) {
    var damage = Math.max(1, attackerAtk - defenderDef + randInt(-1, 2));
    defender.hp -= damage;
    return damage;
}

function playerAttackMonster(player, monster) {
    var atk = player.atk + (player.weapon ? player.weapon.atk : 0);
    var damage = meleeAttack(player, monster, atk, monster.def);
    addMessage(player, 'You hit the ' + monster.name + ' for ' + damage + ' damage.');
    if (monster.hp <= 0) {
        addMessage(player, 'You defeated the ' + monster.name + '!');
        player.xp += monster.xp;
        player.kills++;
        checkLevelUp(player);
    }
    return damage;
}

function monsterAttackPlayer(monster, player) {
    var def = player.def + (player.armor ? player.armor.def : 0);
    var damage = meleeAttack(monster, player, monster.atk, def);
    addMessage(player, 'The ' + monster.name + ' hits you for ' + damage + ' damage!');
    if (player.hp <= 0) {
        player.dead = true;
        addMessage(player, 'You have been slain by the ' + monster.name + '!');
    }
    return damage;
}

// ─── Level Up ──────────────────────────────────────────────────────────────

function checkLevelUp(player) {
    while (player.xp >= player.xpToLevel) {
        player.level++;
        player.xp -= player.xpToLevel;
        player.xpToLevel = Math.floor(player.xpToLevel * 1.5);
        player.maxHp += 5;
        player.hp = Math.min(player.hp + 5, player.maxHp);
        player.atk += 1;
        player.def += 1;
        addMessage(player, 'Welcome to level ' + player.level + '!');
    }
}

// ─── Monster AI ────────────────────────────────────────────────────────────

function updateMonsters(level, players, step) {
    for (var m = 0; m < level.monsters.length; m++) {
        var mon = level.monsters[m];
        if (mon.hp <= 0) continue;
        // Speed check: skip if step not aligned
        if (step % mon.speed !== 0) continue;

        // Find nearest player on this level
        var nearest = null;
        var nearestDist = 999;
        for (var pid in players) {
            var p = players[pid];
            if (p.dead || p.depth !== level.depth) continue;
            var dist = Math.abs(p.x - mon.x) + Math.abs(p.y - mon.y);
            if (dist < nearestDist) {
                nearestDist = dist;
                nearest = p;
            }
        }
        if (!nearest) continue;

        // If adjacent, attack
        if (nearestDist <= 1) {
            monsterAttackPlayer(mon, nearest);
            continue;
        }

        // Chase if within detection range (8 tiles)
        if (nearestDist <= 8) {
            moveMonsterToward(mon, nearest.x, nearest.y, level);
        } else {
            // Wander randomly
            wanderMonster(mon, level);
        }
    }
}

function moveMonsterToward(mon, tx, ty, level) {
    var dx = 0;
    var dy = 0;
    if (tx > mon.x) dx = 1;
    else if (tx < mon.x) dx = -1;
    if (ty > mon.y) dy = 1;
    else if (ty < mon.y) dy = -1;

    // Try primary direction, then alternatives
    var moves = [];
    if (dx !== 0 && dy !== 0) {
        moves.push({ x: dx, y: dy });
        moves.push({ x: dx, y: 0 });
        moves.push({ x: 0, y: dy });
    } else if (dx !== 0) {
        moves.push({ x: dx, y: 0 });
        moves.push({ x: dx, y: 1 });
        moves.push({ x: dx, y: -1 });
    } else {
        moves.push({ x: 0, y: dy });
        moves.push({ x: 1, y: dy });
        moves.push({ x: -1, y: dy });
    }

    for (var i = 0; i < moves.length; i++) {
        var nx = mon.x + moves[i].x;
        var ny = mon.y + moves[i].y;
        if (nx >= 0 && nx < level.width && ny >= 0 && ny < level.height &&
            isWalkable(level.grid[ny][nx]) && !monsterAt(level, nx, ny)) {
            mon.x = nx;
            mon.y = ny;
            return;
        }
    }
}

function wanderMonster(mon, level) {
    var dirs = [{ x: 0, y: -1 }, { x: 0, y: 1 }, { x: -1, y: 0 }, { x: 1, y: 0 }];
    var dir = randPick(dirs);
    var nx = mon.x + dir.x;
    var ny = mon.y + dir.y;
    if (nx >= 0 && nx < level.width && ny >= 0 && ny < level.height &&
        isWalkable(level.grid[ny][nx]) && !monsterAt(level, nx, ny)) {
        mon.x = nx;
        mon.y = ny;
    }
}

function monsterAt(level, x, y) {
    for (var m = 0; m < level.monsters.length; m++) {
        if (level.monsters[m].hp > 0 && level.monsters[m].x === x && level.monsters[m].y === y) {
            return level.monsters[m];
        }
    }
    return null;
}

// ─── Item Interaction ──────────────────────────────────────────────────────

function pickupItem(player, level) {
    for (var i = level.items.length - 1; i >= 0; i--) {
        var item = level.items[i];
        if (item.x === player.x && item.y === player.y) {
            if (item.category === 'gold') {
                player.gold += item.def.value;
                addMessage(player, 'You pick up ' + item.def.value + ' gold.');
                level.items.splice(i, 1);
                return true;
            }
            if (player.inventory.length >= player.maxInventory) {
                addMessage(player, 'Your inventory is full!');
                return false;
            }
            player.inventory.push({ category: item.category, def: item.def });
            addMessage(player, 'You pick up the ' + item.def.name + '.');
            level.items.splice(i, 1);
            return true;
        }
    }
    addMessage(player, 'Nothing to pick up here.');
    return false;
}

function useItem(player, index) {
    if (index < 0 || index >= player.inventory.length) return;
    var item = player.inventory[index];

    switch (item.category) {
        case 'weapons':
            if (player.weapon) {
                player.inventory.push({ category: 'weapons', def: player.weapon });
                addMessage(player, 'You unequip the ' + player.weapon.name + '.');
            }
            player.weapon = item.def;
            player.inventory.splice(index, 1);
            addMessage(player, 'You equip the ' + item.def.name + '.');
            break;
        case 'armor':
            if (player.armor) {
                player.inventory.push({ category: 'armor', def: player.armor });
                addMessage(player, 'You unequip the ' + player.armor.name + '.');
            }
            player.armor = item.def;
            player.inventory.splice(index, 1);
            addMessage(player, 'You equip the ' + item.def.name + '.');
            break;
        case 'potions':
            applyPotion(player, item.def);
            player.inventory.splice(index, 1);
            break;
        case 'scrolls':
            // Scrolls handled in main.js (need access to level)
            break;
        case 'food':
            player.hunger = Math.min(1000, player.hunger + item.def.nutrition);
            addMessage(player, 'You eat the ' + item.def.name + '. Delicious!');
            player.inventory.splice(index, 1);
            break;
    }
}

function applyPotion(player, def) {
    switch (def.effect) {
        case 'heal':
            var healed = Math.min(def.value, player.maxHp - player.hp);
            player.hp += healed;
            addMessage(player, 'You drink the ' + def.name + '. Healed ' + healed + ' HP.');
            break;
        case 'str':
            player.atk += def.value;
            addMessage(player, 'You drink the ' + def.name + '. You feel stronger!');
            break;
        case 'damage':
            player.hp -= def.value;
            addMessage(player, 'You drink the ' + def.name + '. It was poison!');
            if (player.hp <= 0) {
                player.dead = true;
                addMessage(player, 'The poison kills you!');
            }
            break;
    }
}

// ─── Trap Interaction ──────────────────────────────────────────────────────

function checkTraps(player, level) {
    for (var i = 0; i < level.traps.length; i++) {
        var trap = level.traps[i];
        if (trap.x === player.x && trap.y === player.y) {
            if (!trap.revealed) {
                trap.revealed = true;
                player.hp -= trap.def.damage;
                addMessage(player, 'You ' + trap.def.message + '! (-' + trap.def.damage + ' HP)');
                if (player.hp <= 0) {
                    player.dead = true;
                    addMessage(player, 'The ' + trap.def.name + ' kills you!');
                }
                return true;
            }
        }
    }
    return false;
}

// ─── Messages ──────────────────────────────────────────────────────────────

var MAX_MESSAGES = 5;

function addMessage(player, text) {
    player.messages.push(text);
    if (player.messages.length > MAX_MESSAGES) {
        player.messages.shift();
    }
}
