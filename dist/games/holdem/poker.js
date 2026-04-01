// poker.js — Core poker logic: deck, hand evaluation, card rendering

// ─── Card Constants ────────────────────────────────────────────────────
var SUITS = ['s', 'h', 'd', 'c'];
var SUIT_SYMBOLS = { s: '\u2660', h: '\u2665', d: '\u2666', c: '\u2663' };
var SUIT_COLORS  = { s: '\x1b[37m', h: '\x1b[31m', d: '\x1b[31m', c: '\x1b[37m' };
var RANKS = ['2','3','4','5','6','7','8','9','T','J','Q','K','A'];
var RST  = '\x1b[0m';
var BOLD = '\x1b[1m';
var DIM  = '\x1b[2m';
var RED  = '\x1b[31m';
var GRN  = '\x1b[32m';
var YEL  = '\x1b[33m';
var CYN  = '\x1b[36m';
var WHT  = '\x1b[37m';
var FGGOLD = '\x1b[38;5;136m';

var RANK_VAL = {};
for (var _ri = 0; _ri < RANKS.length; _ri++) RANK_VAL[RANKS[_ri]] = _ri + 2;

var HAND_NAMES = [
    'High Card', 'Pair', 'Two Pair', 'Three of a Kind',
    'Straight', 'Flush', 'Full House', 'Four of a Kind',
    'Straight Flush', 'Royal Flush'
];

// ─── Card Rendering ────────────────────────────────────────────────────
function cardStr(card) {
    if (!card) return DIM + '??' + RST;
    var r = card.rank === 'T' ? '10' : card.rank;
    return SUIT_COLORS[card.suit] + BOLD + r + SUIT_SYMBOLS[card.suit] + RST;
}

function cardBack() {
    return '\x1b[34m' + BOLD + '\u2588\u2588' + RST;
}

// Render a card inside brackets: [A♠]
function cardBox(card) {
    return '[' + cardStr(card) + ']';
}

function cardBackBox() {
    return '[' + cardBack() + ']';
}

// ─── Deck ──────────────────────────────────────────────────────────────
function makeDeck() {
    var deck = [];
    for (var si = 0; si < SUITS.length; si++) {
        for (var ri = 0; ri < RANKS.length; ri++) {
            deck.push({ rank: RANKS[ri], suit: SUITS[si] });
        }
    }
    return deck;
}

function shuffle(arr) {
    for (var i = arr.length - 1; i > 0; i--) {
        var j = Math.floor(Math.random() * (i + 1));
        var tmp = arr[i]; arr[i] = arr[j]; arr[j] = tmp;
    }
    return arr;
}

// ─── Hand Evaluation ───────────────────────────────────────────────────
function combinations(arr, k) {
    var result = [];
    function helper(start, combo) {
        if (combo.length === k) { result.push(combo.slice()); return; }
        for (var i = start; i < arr.length; i++) {
            combo.push(arr[i]);
            helper(i + 1, combo);
            combo.pop();
        }
    }
    helper(0, []);
    return result;
}

function uniqueCount(arr) {
    var seen = {};
    var count = 0;
    for (var i = 0; i < arr.length; i++) {
        if (!seen[arr[i]]) { seen[arr[i]] = true; count++; }
    }
    return count;
}

function evalFive(cards) {
    var vals = [], suits = [];
    for (var i = 0; i < cards.length; i++) {
        vals.push(RANK_VAL[cards[i].rank]);
        suits.push(cards[i].suit);
    }
    vals.sort(function(a, b) { return b - a; });

    var isFlush = suits[0] === suits[1] && suits[1] === suits[2] &&
                  suits[2] === suits[3] && suits[3] === suits[4];
    var isStraight = false;
    var straightHigh = 0;
    if (vals[0] - vals[4] === 4 && uniqueCount(vals) === 5) {
        isStraight = true;
        straightHigh = vals[0];
    }
    if (vals[0] === 14 && vals[1] === 5 && vals[2] === 4 && vals[3] === 3 && vals[4] === 2) {
        isStraight = true;
        straightHigh = 5;
    }

    var counts = {};
    for (var i = 0; i < vals.length; i++) {
        counts[vals[i]] = (counts[vals[i]] || 0) + 1;
    }
    var groups = [];
    for (var v in counts) groups.push({ val: parseInt(v), count: counts[v] });
    groups.sort(function(a, b) { return b.count - a.count || b.val - a.val; });

    if (isStraight && isFlush) {
        if (straightHigh === 14) return { rank: 9, kickers: [14], name: 'Royal Flush' };
        return { rank: 8, kickers: [straightHigh], name: 'Straight Flush' };
    }
    if (groups[0].count === 4) return { rank: 7, kickers: [groups[0].val, groups[1].val], name: 'Four of a Kind' };
    if (groups[0].count === 3 && groups[1].count === 2) return { rank: 6, kickers: [groups[0].val, groups[1].val], name: 'Full House' };
    if (isFlush) return { rank: 5, kickers: vals, name: 'Flush' };
    if (isStraight) return { rank: 4, kickers: [straightHigh], name: 'Straight' };
    if (groups[0].count === 3) return { rank: 3, kickers: [groups[0].val, groups[1].val, groups[2].val], name: 'Three of a Kind' };
    if (groups[0].count === 2 && groups[1].count === 2) {
        var hi = Math.max(groups[0].val, groups[1].val);
        var lo = Math.min(groups[0].val, groups[1].val);
        return { rank: 2, kickers: [hi, lo, groups[2].val], name: 'Two Pair' };
    }
    if (groups[0].count === 2) return { rank: 1, kickers: [groups[0].val, groups[1].val, groups[2].val, groups[3].val], name: 'Pair' };
    return { rank: 0, kickers: vals, name: 'High Card' };
}

function evaluateHand(cards) {
    var best = null;
    var combos = combinations(cards, 5);
    for (var ci = 0; ci < combos.length; ci++) {
        var h = evalFive(combos[ci]);
        if (!best || compareHands(h, best) > 0) best = h;
    }
    return best;
}

function compareHands(a, b) {
    if (a.rank !== b.rank) return a.rank - b.rank;
    for (var i = 0; i < Math.min(a.kickers.length, b.kickers.length); i++) {
        if (a.kickers[i] !== b.kickers[i]) return a.kickers[i] - b.kickers[i];
    }
    return 0;
}

// ─── AI Hand Strength Heuristic ────────────────────────────────────────
function aiHandStrength(hand, community) {
    if (!hand || hand.length < 2) return 0.3;
    var c1 = RANK_VAL[hand[0].rank];
    var c2 = RANK_VAL[hand[1].rank];
    var hi = Math.max(c1, c2);
    var lo = Math.min(c1, c2);
    var suited = hand[0].suit === hand[1].suit;
    var pair = c1 === c2;

    var strength = (hi + lo - 4) / 24;
    if (pair) strength += 0.25 + (hi - 2) * 0.02;
    if (suited) strength += 0.06;
    if (hi - lo <= 4 && !pair) strength += 0.04;
    if (hi >= 12) strength += 0.08;

    if (community && community.length >= 3) {
        var allCards = hand.concat(community);
        var eval_ = evaluateHand(allCards);
        strength = 0.2 + eval_.rank * 0.09;
        if (eval_.rank >= 2) strength += 0.1;
        if (eval_.rank >= 4) strength += 0.15;
    }
    return Math.min(1.0, Math.max(0.0, strength));
}
