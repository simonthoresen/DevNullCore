// items.js — Item definitions, loot tables, and spawning

// ─── Item Definitions ──────────────────────────────────────────────────────

var WEAPONS = [
    { name: 'Rusty Sword',   slot: 'mainHand', atk: 2,  tier: 1 },
    { name: 'Iron Sword',    slot: 'mainHand', atk: 4,  tier: 2 },
    { name: 'Steel Blade',   slot: 'mainHand', atk: 7,  tier: 3 },
    { name: 'War Axe',       slot: 'mainHand', atk: 6,  tier: 2 },
    { name: 'Mace',          slot: 'mainHand', atk: 5,  tier: 2 },
    { name: 'Flaming Sword', slot: 'mainHand', atk: 10, tier: 4 },
    { name: 'Dagger',        slot: 'mainHand', atk: 3,  tier: 1 },
    { name: 'Halberd',       slot: 'mainHand', atk: 8,  tier: 3 }
];

var ARMOR_HEAD = [
    { name: 'Leather Cap',   slot: 'head', def: 1, tier: 1 },
    { name: 'Iron Helm',     slot: 'head', def: 3, tier: 2 },
    { name: 'Steel Helm',    slot: 'head', def: 5, tier: 3 }
];

var ARMOR_CHEST = [
    { name: 'Leather Vest',  slot: 'chest', def: 2, tier: 1 },
    { name: 'Chainmail',     slot: 'chest', def: 4, tier: 2 },
    { name: 'Plate Armor',   slot: 'chest', def: 7, tier: 3 },
    { name: 'Dragon Mail',   slot: 'chest', def: 10, tier: 4 }
];

var ARMOR_LEGS = [
    { name: 'Cloth Pants',   slot: 'legs', def: 1, tier: 1 },
    { name: 'Iron Greaves',  slot: 'legs', def: 3, tier: 2 },
    { name: 'Steel Greaves', slot: 'legs', def: 5, tier: 3 }
];

var ARMOR_FEET = [
    { name: 'Sandals',       slot: 'feet', def: 1, tier: 1 },
    { name: 'Leather Boots', slot: 'feet', def: 2, tier: 1 },
    { name: 'Iron Boots',    slot: 'feet', def: 3, tier: 2 },
    { name: 'Greaves',       slot: 'feet', def: 5, tier: 3 }
];

var SHIELDS = [
    { name: 'Buckler',       slot: 'offHand', def: 1, tier: 1 },
    { name: 'Round Shield',  slot: 'offHand', def: 3, tier: 2 },
    { name: 'Tower Shield',  slot: 'offHand', def: 5, tier: 3 }
];

var RINGS = [
    { name: 'Ring of Power', slot: 'ring', atk: 2,  tier: 2 },
    { name: 'Ring of Guard', slot: 'ring', def: 2,  tier: 2 },
    { name: 'Ruby Ring',     slot: 'ring', atk: 4,  tier: 3 }
];

var AMULETS = [
    { name: 'Health Amulet', slot: 'amulet', def: 1, tier: 1 },
    { name: 'Ward Amulet',   slot: 'amulet', def: 3, tier: 2 },
    { name: 'Dragon Amulet', slot: 'amulet', def: 5, tier: 3 }
];

var CONSUMABLES = [
    { name: 'Health Potion',    type: 'consumable', heal: 25, tier: 1 },
    { name: 'Greater Potion',   type: 'consumable', heal: 50, tier: 2 },
    { name: 'Elixir of Life',   type: 'consumable', heal: 100, tier: 3 }
];

var ALL_EQUIPPABLE = [].concat(WEAPONS, ARMOR_HEAD, ARMOR_CHEST, ARMOR_LEGS, ARMOR_FEET, SHIELDS, RINGS, AMULETS);
var ALL_ITEMS = [].concat(ALL_EQUIPPABLE, CONSUMABLES);

// ─── Monster Definitions ───────────────────────────────────────────────────

var MONSTER_DEFS = [
    { name: 'Rat',       ch: 'r', color: '#886644', atk: 2,  hp: 8,   def: 0, xpReward: 3,  tier: 1, loot: 'low' },
    { name: 'Bat',       ch: 'b', color: '#664422', atk: 3,  hp: 6,   def: 0, xpReward: 3,  tier: 1, loot: 'low' },
    { name: 'Goblin',    ch: 'g', color: '#44aa44', atk: 4,  hp: 15,  def: 1, xpReward: 5,  tier: 1, loot: 'low' },
    { name: 'Skeleton',  ch: 's', color: '#cccccc', atk: 5,  hp: 20,  def: 2, xpReward: 8,  tier: 2, loot: 'mid' },
    { name: 'Orc',       ch: 'o', color: '#448844', atk: 7,  hp: 30,  def: 3, xpReward: 12, tier: 2, loot: 'mid' },
    { name: 'Troll',     ch: 'T', color: '#668866', atk: 9,  hp: 45,  def: 4, xpReward: 18, tier: 3, loot: 'mid' },
    { name: 'Wraith',    ch: 'W', color: '#8844aa', atk: 10, hp: 35,  def: 3, xpReward: 20, tier: 3, loot: 'high' },
    { name: 'Ogre',      ch: 'O', color: '#aa8844', atk: 12, hp: 60,  def: 5, xpReward: 25, tier: 3, loot: 'high' },
    { name: 'Dragon',    ch: 'D', color: '#ff4444', atk: 15, hp: 100, def: 8, xpReward: 50, tier: 4, loot: 'high' },
    { name: 'Lich',      ch: 'L', color: '#aa44ff', atk: 18, hp: 80,  def: 6, xpReward: 45, tier: 4, loot: 'high' }
];

// ─── Loot System ───────────────────────────────────────────────────────────

function rollLoot(table, floor) {
    var maxTier = Math.min(4, Math.ceil(floor / 2));
    var pool;

    if (table === 'low') {
        pool = CONSUMABLES.filter(function(i) { return i.tier <= maxTier; });
        if (Math.random() < 0.3) {
            pool = pool.concat(ALL_EQUIPPABLE.filter(function(i) { return i.tier <= Math.max(1, maxTier - 1); }));
        }
    } else if (table === 'mid') {
        pool = ALL_ITEMS.filter(function(i) { return i.tier <= maxTier; });
    } else { // high
        pool = ALL_ITEMS.filter(function(i) { return i.tier >= Math.max(1, maxTier - 1) && i.tier <= maxTier; });
    }

    if (pool.length === 0) return null;
    if (Math.random() < 0.4) return null; // 40% chance of no drop

    // Clone the item so each drop is independent
    var template = pool[Math.floor(Math.random() * pool.length)];
    var item = {};
    for (var k in template) item[k] = template[k];
    return item;
}

// ─── Spawning ──────────────────────────────────────────────────────────────

function scatterItems(maze, floor) {
    var items = [];
    var maxTier = Math.min(4, Math.ceil(floor / 2));
    var count = 3 + Math.floor(Math.random() * 4) + floor;
    var pool = ALL_ITEMS.filter(function(i) { return i.tier <= maxTier; });

    for (var i = 0; i < count && pool.length > 0; i++) {
        var pos = randomFloor(maze);
        if (pos) {
            var template = pool[Math.floor(Math.random() * pool.length)];
            var item = {};
            for (var k in template) item[k] = template[k];
            items.push({ x: pos.x, y: pos.y, item: item });
        }
    }
    return items;
}

function spawnMonsters(maze, floor) {
    var monsters = [];
    var maxTier = Math.min(4, Math.ceil(floor / 2));
    var pool = MONSTER_DEFS.filter(function(m) { return m.tier <= maxTier; });
    var count = 4 + Math.floor(Math.random() * 3) + Math.floor(floor * 1.5);

    for (var i = 0; i < count && pool.length > 0; i++) {
        var pos = randomFloor(maze);
        if (pos && !(pos.x === maze.startX && pos.y === maze.startY)) {
            var def = pool[Math.floor(Math.random() * pool.length)];
            monsters.push({
                x: pos.x,
                y: pos.y,
                name: def.name,
                ch: def.ch,
                color: def.color,
                hp: def.hp + Math.floor(floor * 2),
                maxHp: def.hp + Math.floor(floor * 2),
                atk: def.atk + Math.floor(floor * 0.5),
                def: def.def,
                xpReward: def.xpReward,
                loot: def.loot
            });
        }
    }
    return monsters;
}

function randomFloor(maze) {
    for (var attempts = 0; attempts < 100; attempts++) {
        var x = 1 + Math.floor(Math.random() * (maze.w - 2));
        var y = 1 + Math.floor(Math.random() * (maze.h - 2));
        if (maze.grid[y][x] === TILE_FLOOR) {
            return { x: x, y: y };
        }
    }
    return null;
}
