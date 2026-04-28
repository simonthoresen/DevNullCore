// ui.js — NC widget tree rendering for Texas Hold'em
// Returns declarative widget trees that the framework renders using real NC controls.

// ─── Main View (returns widget tree for viewNC) ────────────────────────

function buildViewNC(teamID, width, height) {
    var seat = _s.seats[teamID];

    var playersW = Math.min(30, Math.floor(width * 0.35));

    return {
        type: 'hsplit',
        children: [
            {
                type: 'vsplit', weight: 1,
                children: [
                    buildTablePanel(teamID),
                    buildHandPanel(teamID)
                ]
            },
            buildPlayersPanel(teamID, playersW)
        ]
    };
}

// ─── Table Panel (community cards + pot) ───────────────────────────────

function buildTablePanel(teamID) {
    var children = [];

    // Phase
    var phaseStr = '';
    if (_s.phase === 'waiting') phaseStr = DIM + 'Waiting for players...' + RST;
    else if (_s.phase === 'preflop') phaseStr = 'Pre-Flop';
    else if (_s.phase === 'flop') phaseStr = 'Flop';
    else if (_s.phase === 'turn') phaseStr = 'Turn';
    else if (_s.phase === 'river') phaseStr = 'River';
    else if (_s.phase === 'showdown') phaseStr = YEL + BOLD + 'Showdown!' + RST;
    else if (_s.phase === 'between') phaseStr = DIM + (_s.lastWinMsg || 'Next hand...') + RST;

    children.push({ type: 'label', text: phaseStr, align: 'center', height: 1 });
    children.push({ type: 'label', text: '', height: 1 });

    // Community cards
    var commStr = '';
    if (_s.phase === 'waiting') {
        commStr = DIM + '[  ] [  ] [  ]  [  ]  [  ]' + RST;
    } else {
        for (var ci = 0; ci < 5; ci++) {
            if (ci > 0) commStr += ' ';
            if (ci < _s.community.length) {
                commStr += cardBox(_s.community[ci]);
            } else {
                commStr += DIM + '[  ]' + RST;
            }
        }
    }
    children.push({ type: 'label', text: commStr, align: 'center', height: 1 });
    children.push({ type: 'label', text: '', height: 1 });

    // Pot
    if (_s.phase !== 'waiting') {
        children.push({ type: 'label', text: FGGOLD + BOLD + 'Pot: ' + _s.pot + RST, align: 'center', height: 1 });
    }

    // Last action
    if (_s.lastAction) {
        children.push({ type: 'label', text: DIM + _s.lastAction + RST, align: 'center', height: 1 });
    }

    // Showdown results
    if (_s.phase === 'showdown' && _s.showdownResults) {
        children.push({ type: 'label', text: '', height: 1 });
        for (var ri = 0; ri < _s.showdownResults.length; ri++) {
            var sr = _s.showdownResults[ri];
            var srLine = sr.name + ': ' + cardBox(sr.cards[0]) + cardBox(sr.cards[1]);
            srLine += ' ' + GRN + sr.hand.name + RST;
            children.push({ type: 'label', text: srLine, align: 'center', height: 1 });
        }
    }

    var title = 'Hand #' + _s.handNum + '  Blinds ' + _s.SMALL_BLIND + '/' + _s.BIG_BLIND;
    if (_s.phase === 'waiting') title = "Texas Hold'em";

    return {
        type: 'panel', title: title, weight: 1,
        children: children
    };
}

// ─── Hand Panel (your team's cards) ────────────────────────────────────

function buildHandPanel(teamID) {
    var seat = _s.seats[teamID];
    var children = [];

    if (!seat) {
        children.push({ type: 'label', text: DIM + 'Spectating' + RST, align: 'center' });
    } else if (seat.bustedOut) {
        children.push({ type: 'label', text: DIM + 'Eliminated' + RST, align: 'center' });
    } else if (seat.hand.length === 2) {
        children.push({ type: 'label', text: '  ' + cardBox(seat.hand[0]) + '  ' + cardBox(seat.hand[1]) + '  ', align: 'center' });

        if (_s.community.length >= 3) {
            var allCards = seat.hand.concat(_s.community);
            var eval_ = evaluateHand(allCards);
            children.push({ type: 'label', text: GRN + eval_.name + RST, align: 'center' });
        }

        var chipLine = FGGOLD + '$' + seat.chips + RST;
        if (seat.bet > 0) chipLine += '  ' + RED + 'Bet: ' + seat.bet + RST;
        if (seat.allIn) chipLine += '  ' + YEL + BOLD + 'ALL-IN' + RST;
        children.push({ type: 'label', text: chipLine, align: 'center' });
    } else if (seat.folded) {
        children.push({ type: 'label', text: DIM + 'Folded' + RST, align: 'center' });
    } else {
        children.push({ type: 'label', text: DIM + 'Waiting for deal...' + RST, align: 'center' });
    }

    return {
        type: 'panel', title: 'Your Hand', height: 7,
        children: children
    };
}

// ─── Players Panel (all seats) ─────────────────────────────────────────

function buildPlayersPanel(teamID, panelWidth) {
    var rows = [];

    for (var si = 0; si < _s.seatOrder.length; si++) {
        var sid = _s.seatOrder[si];
        var seat = _s.seats[sid];
        if (!seat) continue;

        var isMe = sid === teamID;
        var isDealer = _s.dealerIdx === si;
        var isAction = _s.actionOn === si;

        // Name with indicators
        var nameStr = '';
        if (isDealer) nameStr += YEL + 'D ' + RST;
        if (isAction && _s.phase !== 'waiting' && _s.phase !== 'showdown' && _s.phase !== 'between') {
            nameStr += WHT + BOLD + '\u25b6 ' + RST;
        }
        var displayName = seat.name;
        if (seat.isAI) displayName += DIM + ' bot' + RST;
        nameStr += (isMe ? CYN + BOLD : WHT) + displayName + RST;

        if (seat.bustedOut) nameStr = DIM + '\u2620 ' + seat.name + RST;
        else if (seat.folded && _s.phase !== 'waiting' && _s.phase !== 'between') {
            nameStr = DIM + seat.name + ' (fold)' + RST;
        }

        // Chips + bet
        var chipStr = FGGOLD + '$' + seat.chips + RST;
        if (seat.bet > 0) chipStr += ' ' + RED + '+' + seat.bet + RST;
        if (seat.allIn) chipStr += ' ' + YEL + 'AI' + RST;

        // Cards
        var cardLine = '';
        if (_s.phase === 'showdown' && !seat.folded && seat.hand.length === 2) {
            cardLine = cardBox(seat.hand[0]) + cardBox(seat.hand[1]);
        } else if (isMe && seat.hand.length === 2) {
            cardLine = cardBox(seat.hand[0]) + cardBox(seat.hand[1]);
        } else if (seat.hand.length === 2 && !seat.folded && !seat.bustedOut) {
            cardLine = cardBackBox() + cardBackBox();
        }

        rows.push([nameStr, chipStr, cardLine]);
    }

    return {
        type: 'panel', title: 'Players', width: panelWidth,
        children: [{
            type: 'table',
            rows: rows
        }]
    };
}

// ─── Status Bar ────────────────────────────────────────────────────────

function renderStatusBar(teamID) {
    var seat = _s.seats[teamID];
    var chips = seat ? '$' + seat.chips : '';
    var phase = _s.phase === 'waiting' ? 'Waiting' :
                _s.phase.charAt(0).toUpperCase() + _s.phase.slice(1);
    var nPlayers = activePlayers().length;
    return "Hold'em  |  " + phase + '  |  Blinds ' + _s.SMALL_BLIND + '/' + _s.BIG_BLIND +
           '  |  ' + chips + '  |  ' + nPlayers + ' seats';
}

// ─── Command Bar ───────────────────────────────────────────────────────

function renderCommandBar(teamID) {
    var seat = _s.seats[teamID];
    if (!seat) return '[Enter] Chat';

    if (_s.phase === 'waiting') {
        var need = 2 - activePlayers().length;
        if (need > 0) return 'Waiting for ' + need + ' more team(s)...  [Enter] Chat';
        return 'Starting soon...  [Enter] Chat';
    }
    if (_s.phase === 'showdown' || _s.phase === 'between') return '[Enter] Chat';
    if (seat.bustedOut) return 'Eliminated  [Enter] Chat';
    if (seat.folded) return 'Folded  [Enter] Chat';

    if (_s.actionOn >= 0 && _s.seatOrder[_s.actionOn] === teamID) {
        var toCall = _s.currentBet - seat.bet;
        var actions = '';
        if (toCall > 0) {
            actions = '[Space/C] Call ' + toCall + '  [F] Fold';
        } else {
            actions = '[Space/C] Check';
        }
        if (seat.chips > toCall) {
            actions += '  [R] Raise ' + _s.raiseAmount + '  [\u2191\u2193] Adjust';
        }
        actions += '  [A] All-in ' + seat.chips;
        return actions;
    }

    var waitName = '...';
    if (_s.actionOn >= 0 && _s.seats[_s.seatOrder[_s.actionOn]]) {
        waitName = _s.seats[_s.seatOrder[_s.actionOn]].name;
    }
    return 'Waiting for ' + waitName + '  [Enter] Chat';
}
