// data.js — Monster definitions, item tables, and tile characters for nethack
// Inspired by classic NetHack but simplified for multiplayer terminal play.

// ─── Tile Characters ───────────────────────────────────────────────────────
var TILES = {
    WALL:       '#',
    FLOOR:      '.',
    CORRIDOR:   '.',
    DOOR_OPEN:  '/',
    DOOR_CLOSED:'+',
    STAIRS_DOWN:'>',
    STAIRS_UP:  '<',
    VOID:       ' ',
    WATER:      '~',
    TRAP:       '^'
};

// ─── Monster Definitions ───────────────────────────────────────────────────
// Each entry: { ch, name, hp, atk, def, speed, xp, minDepth }
// speed: 1 = every step, 2 = every other step, etc.
var MONSTER_DEFS = [
    { ch: 'r', name: 'rat',             hp: 4,   atk: 2,  def: 0,  speed: 1, xp: 5,   minDepth: 1 },
    { ch: 'b', name: 'bat',             hp: 3,   atk: 1,  def: 0,  speed: 1, xp: 4,   minDepth: 1 },
    { ch: 'k', name: 'kobold',          hp: 6,   atk: 3,  def: 1,  speed: 2, xp: 10,  minDepth: 1 },
    { ch: 'g', name: 'goblin',          hp: 8,   atk: 4,  def: 1,  speed: 2, xp: 15,  minDepth: 2 },
    { ch: 'j', name: 'jackal',          hp: 5,   atk: 3,  def: 0,  speed: 1, xp: 8,   minDepth: 1 },
    { ch: 'G', name: 'gnome',           hp: 10,  atk: 5,  def: 2,  speed: 2, xp: 20,  minDepth: 2 },
    { ch: 'o', name: 'orc',             hp: 14,  atk: 6,  def: 3,  speed: 2, xp: 30,  minDepth: 3 },
    { ch: 's', name: 'snake',           hp: 7,   atk: 4,  def: 1,  speed: 1, xp: 12,  minDepth: 2 },
    { ch: 'Z', name: 'zombie',          hp: 18,  atk: 5,  def: 2,  speed: 3, xp: 25,  minDepth: 3 },
    { ch: 'S', name: 'skeleton',        hp: 15,  atk: 7,  def: 4,  speed: 2, xp: 35,  minDepth: 4 },
    { ch: 'h', name: 'hobgoblin',       hp: 20,  atk: 8,  def: 4,  speed: 2, xp: 40,  minDepth: 4 },
    { ch: 'w', name: 'wolf',            hp: 12,  atk: 6,  def: 2,  speed: 1, xp: 22,  minDepth: 3 },
    { ch: 'H', name: 'hill giant',      hp: 35,  atk: 12, def: 6,  speed: 3, xp: 80,  minDepth: 6 },
    { ch: 'O', name: 'ogre',            hp: 30,  atk: 10, def: 5,  speed: 3, xp: 60,  minDepth: 5 },
    { ch: 'T', name: 'troll',           hp: 40,  atk: 14, def: 5,  speed: 2, xp: 100, minDepth: 7 },
    { ch: 'W', name: 'wraith',          hp: 25,  atk: 10, def: 3,  speed: 2, xp: 70,  minDepth: 6 },
    { ch: 'V', name: 'vampire',         hp: 35,  atk: 12, def: 5,  speed: 1, xp: 90,  minDepth: 7 },
    { ch: 'D', name: 'dragon',          hp: 60,  atk: 18, def: 10, speed: 2, xp: 200, minDepth: 9 },
    { ch: 'L', name: 'lich',            hp: 45,  atk: 15, def: 8,  speed: 2, xp: 150, minDepth: 8 },
    { ch: '&', name: 'demon',           hp: 55,  atk: 16, def: 9,  speed: 1, xp: 180, minDepth: 9 }
];

// ─── Item Definitions ──────────────────────────────────────────────────────
// Types: weapon, armor, potion, scroll, food, gold
var ITEM_DEFS = {
    weapons: [
        { ch: ')', name: 'dagger',         atk: 3,  minDepth: 1 },
        { ch: ')', name: 'short sword',    atk: 5,  minDepth: 2 },
        { ch: ')', name: 'mace',           atk: 6,  minDepth: 3 },
        { ch: ')', name: 'long sword',     atk: 8,  minDepth: 4 },
        { ch: ')', name: 'battle axe',     atk: 10, minDepth: 5 },
        { ch: ')', name: 'two-handed sword',atk: 12, minDepth: 7 }
    ],
    armor: [
        { ch: '[', name: 'leather armor',  def: 2,  minDepth: 1 },
        { ch: '[', name: 'ring mail',      def: 3,  minDepth: 2 },
        { ch: '[', name: 'chain mail',     def: 5,  minDepth: 3 },
        { ch: '[', name: 'plate mail',     def: 7,  minDepth: 5 },
        { ch: '[', name: 'crystal armor',  def: 9,  minDepth: 7 }
    ],
    potions: [
        { ch: '!', name: 'potion of healing',       effect: 'heal',   value: 15, minDepth: 1 },
        { ch: '!', name: 'potion of greater healing',effect: 'heal',  value: 30, minDepth: 4 },
        { ch: '!', name: 'potion of strength',      effect: 'str',    value: 2,  minDepth: 3 },
        { ch: '!', name: 'potion of poison',        effect: 'damage', value: 10, minDepth: 2 }
    ],
    scrolls: [
        { ch: '?', name: 'scroll of teleport',      effect: 'teleport', minDepth: 2 },
        { ch: '?', name: 'scroll of mapping',       effect: 'map',      minDepth: 3 },
        { ch: '?', name: 'scroll of identify',      effect: 'identify', minDepth: 1 }
    ],
    food: [
        { ch: '%', name: 'ration',         nutrition: 200, minDepth: 1 },
        { ch: '%', name: 'apple',          nutrition: 80,  minDepth: 1 },
        { ch: '%', name: 'meat',           nutrition: 150, minDepth: 2 }
    ]
};

// ─── Trap Definitions ──────────────────────────────────────────────────────
var TRAP_DEFS = [
    { name: 'pit trap',   damage: 5,  message: 'fall into a pit' },
    { name: 'dart trap',  damage: 8,  message: 'are hit by a dart' },
    { name: 'bear trap',  damage: 10, message: 'step into a bear trap' }
];

// ─── Helper: get monsters eligible for a given depth ───────────────────────
function getMonstersForDepth(depth) {
    var result = [];
    for (var i = 0; i < MONSTER_DEFS.length; i++) {
        if (MONSTER_DEFS[i].minDepth <= depth) {
            result.push(MONSTER_DEFS[i]);
        }
    }
    return result;
}

// ─── Helper: get items eligible for a given depth ──────────────────────────
function getItemsForDepth(depth) {
    var result = [];
    var categories = ['weapons', 'armor', 'potions', 'scrolls', 'food'];
    for (var c = 0; c < categories.length; c++) {
        var items = ITEM_DEFS[categories[c]];
        for (var i = 0; i < items.length; i++) {
            if (items[i].minDepth <= depth) {
                result.push({ category: categories[c], def: items[i] });
            }
        }
    }
    return result;
}
