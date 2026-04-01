// ui.js — NC-panel based rendering for Texas Hold'em
// Uses ncui.js for box-drawing panel primitives.

// ─── Layout Constants ──────────────────────────────────────────────────
var PLAYERS_PANEL_W = 30;
var HAND_PANEL_H = 7;

// ─── Main View ─────────────────────────────────────────────────────────

function renderGameView(teamID, width, height) {
    var seat = state.seats[teamID];

    // Calculate panel widths
    var playersW = Math.min(PLAYERS_PANEL_W, Math.floor(width * 0.35));
    var mainW = width - playersW;
    var mainH = height;

    // Left side: table panel (top) + hand panel (bottom)
    var handH = HAND_PANEL_H;
    var tableH = mainH - handH;
    if (tableH < 6) { tableH = mainH; handH = 0; }

    var tablePanel = renderTablePanel(teamID, mainW, tableH);
    var handPanel = handH > 0 ? renderHandPanel(teamID, mainW, handH) : [];
    var leftLines = tablePanel.concat(handPanel);

    // Right side: players panel
    var playersPanel = renderPlayersPanel(teamID, playersW, mainH);

    // Merge side by side
    var merged = ncHConcat(leftLines, playersPanel);
    return ncFillScreen(merged, width, height).join('\n');
}

// ─── Table Panel (community cards + pot) ───────────────────────────────

function renderTablePanel(teamID, width, height) {
    var innerW = width - 2;
    var lines = [];

    // Phase
    var phaseStr = '';
    if (state.phase === 'waiting') phaseStr = DIM + 'Waiting for players...' + RST;
    else if (state.phase === 'preflop') phaseStr = 'Pre-Flop';
    else if (state.phase === 'flop') phaseStr = 'Flop';
    else if (state.phase === 'turn') phaseStr = 'Turn';
    else if (state.phase === 'river') phaseStr = 'River';
    else if (state.phase === 'showdown') phaseStr = YEL + BOLD + 'Showdown!' + RST;
    else if (state.phase === 'between') phaseStr = DIM + (state.lastWinMsg || 'Next hand...') + RST;

    lines.push(ncCenter(phaseStr, innerW));
    lines.push('');

    // Community cards
    var commStr = '';
    if (state.phase === 'waiting') {
        commStr = DIM + '[  ] [  ] [  ]  [  ]  [  ]' + RST;
    } else {
        for (var ci = 0; ci < 5; ci++) {
            if (ci > 0) commStr += ' ';
            if (ci < state.community.length) {
                commStr += cardBox(state.community[ci]);
            } else {
                commStr += DIM + '[  ]' + RST;
            }
        }
    }
    lines.push(ncCenter(commStr, innerW));
    lines.push('');

    // Pot
    if (state.phase !== 'waiting') {
        var potStr = FGGOLD + BOLD + 'Pot: ' + state.pot + RST;
        lines.push(ncCenter(potStr, innerW));
    } else {
        lines.push('');
    }

    // Last action
    if (state.lastAction) {
        lines.push('');
        lines.push(ncCenter(DIM + state.lastAction + RST, innerW));
    }

    // Action timer for current team
    var actionTeam = state.actionOn >= 0 ? state.seatOrder[state.actionOn] : null;
    if (actionTeam === teamID && state.phase !== 'waiting' && state.phase !== 'showdown' && state.phase !== 'between') {
        var seat = state.seats[teamID];
        if (seat && !seat.folded && !seat.allIn) {
            lines.push('');
            var pct = state.actionTimer / ACTION_TIMEOUT;
            var barW = Math.min(20, innerW - 6);
            var timerColor = pct > 0.5 ? GRN : pct > 0.2 ? YEL : RED;
            var secs = Math.ceil(state.actionTimer / 10);
            var bar = timerColor + ncProgressBar(barW, pct) + RST + ' ' + timerColor + secs + 's' + RST;
            lines.push(ncCenter(bar, innerW));
        }
    }

    // Showdown results
    if (state.phase === 'showdown' && state.showdownResults) {
        lines.push('');
        for (var ri = 0; ri < state.showdownResults.length; ri++) {
            var sr = state.showdownResults[ri];
            var srLine = sr.name + ': ' + cardBox(sr.cards[0]) + cardBox(sr.cards[1]);
            srLine += ' ' + GRN + sr.hand.name + RST;
            lines.push(ncCenter(srLine, innerW));
        }
    }

    var title = 'Hand #' + state.handNum + '  Blinds ' + SMALL_BLIND + '/' + BIG_BLIND;
    if (state.phase === 'waiting') title = 'Texas Hold\'em';

    return ncPanelWithHeader(width, height, ncCenter(title, innerW), lines);
}

// ─── Hand Panel (your team's cards) ────────────────────────────────────

function renderHandPanel(teamID, width, height) {
    var seat = state.seats[teamID];
    var innerW = width - 2;
    var lines = [];

    if (!seat) {
        lines.push(ncCenter(DIM + 'Spectating' + RST, innerW));
    } else if (seat.bustedOut) {
        lines.push(ncCenter(DIM + 'Eliminated' + RST, innerW));
    } else if (seat.hand.length === 2) {
        // Show cards
        var cardsStr = '  ' + cardBox(seat.hand[0]) + '  ' + cardBox(seat.hand[1]) + '  ';
        lines.push(ncCenter(cardsStr, innerW));

        // Show hand evaluation if community cards are out
        if (state.community.length >= 3) {
            var allCards = seat.hand.concat(state.community);
            var eval_ = evaluateHand(allCards);
            lines.push(ncCenter(GRN + eval_.name + RST, innerW));
        }

        // Show chips and current bet
        var chipLine = FGGOLD + '$' + seat.chips + RST;
        if (seat.bet > 0) chipLine += '  ' + RED + 'Bet: ' + seat.bet + RST;
        if (seat.allIn) chipLine += '  ' + YEL + BOLD + 'ALL-IN' + RST;
        lines.push(ncCenter(chipLine, innerW));
    } else if (seat.folded) {
        lines.push(ncCenter(DIM + 'Folded' + RST, innerW));
    } else {
        lines.push(ncCenter(DIM + 'Waiting for deal...' + RST, innerW));
    }

    // Team member info
    var teamData = getTeamForSeat(teamID);
    if (teamData && teamData.players.length > 1) {
        var memberStr = DIM + 'Team: ' + teamData.name + ' (' + teamData.players.length + ' players)' + RST;
        lines.push(ncCenter(memberStr, innerW));
    }

    return ncPanel(width, height, lines, { title: 'Your Hand', double: false });
}

// ─── Players Panel (all seats) ─────────────────────────────────────────

function renderPlayersPanel(teamID, width, height) {
    var innerW = width - 2;
    var lines = [];

    for (var si = 0; si < state.seatOrder.length; si++) {
        var sid = state.seatOrder[si];
        var seat = state.seats[sid];
        if (!seat) continue;

        var isMe = sid === teamID;
        var isDealer = state.dealerIdx === si;
        var isAction = state.actionOn === si;

        // Name with indicators
        var nameStr = '';
        if (isDealer) nameStr += YEL + 'D ' + RST;
        if (isAction && state.phase !== 'waiting' && state.phase !== 'showdown' && state.phase !== 'between') {
            nameStr += WHT + BOLD + '\u25b6 ' + RST;
        }

        var displayName = seat.name;
        if (seat.isAI) displayName += DIM + ' bot' + RST;
        nameStr += (isMe ? CYN + BOLD : WHT) + displayName + RST;

        if (seat.bustedOut) {
            nameStr = DIM + '\u2620 ' + seat.name + RST;
        } else if (seat.folded && state.phase !== 'waiting' && state.phase !== 'between') {
            nameStr = DIM + seat.name + ' (fold)' + RST;
        }

        // Chips + bet
        var chipStr = FGGOLD + '$' + seat.chips + RST;
        if (seat.bet > 0) chipStr += ' ' + RED + '+' + seat.bet + RST;
        if (seat.allIn) chipStr += ' ' + YEL + 'AI' + RST;

        // Cards (for showdown or own team)
        var cardLine = '';
        if (state.phase === 'showdown' && !seat.folded && seat.hand.length === 2) {
            cardLine = cardBox(seat.hand[0]) + cardBox(seat.hand[1]);
        } else if (isMe && seat.hand.length === 2) {
            cardLine = cardBox(seat.hand[0]) + cardBox(seat.hand[1]);
        } else if (seat.hand.length === 2 && !seat.folded && !seat.bustedOut) {
            cardLine = cardBackBox() + cardBackBox();
        }

        lines.push(ncFit(nameStr, innerW));
        var detailLine = chipStr;
        if (cardLine) detailLine += '  ' + cardLine;
        lines.push(ncFit(' ' + detailLine, innerW));

        // Divider between players (except after last)
        if (si < state.seatOrder.length - 1) {
            lines.push(ncRepeat(NC.IH, innerW));
        }
    }

    return ncPanelWithHeader(width, height,
        ncCenter(activePlayers().length + ' players', innerW),
        lines,
        { title: 'Players' });
}

// ─── Status Bar ────────────────────────────────────────────────────────

function renderStatusBar(teamID) {
    var seat = state.seats[teamID];
    var chips = seat ? '$' + seat.chips : '';
    var phase = state.phase === 'waiting' ? 'Waiting' :
                state.phase.charAt(0).toUpperCase() + state.phase.slice(1);
    var nPlayers = activePlayers().length;
    return "Hold'em  |  " + phase + '  |  Blinds ' + SMALL_BLIND + '/' + BIG_BLIND +
           '  |  ' + chips + '  |  ' + nPlayers + ' seats';
}

// ─── Command Bar ───────────────────────────────────────────────────────

function renderCommandBar(teamID) {
    var seat = state.seats[teamID];
    if (!seat) return '[Enter] Chat';

    if (state.phase === 'waiting') {
        var need = 2 - activePlayers().length;
        if (need > 0) return 'Waiting for ' + need + ' more team(s)...  [Enter] Chat';
        return 'Starting soon...  [Enter] Chat';
    }
    if (state.phase === 'showdown' || state.phase === 'between') return '[Enter] Chat';
    if (seat.bustedOut) return 'Eliminated  [Enter] Chat';
    if (seat.folded) return 'Folded  [Enter] Chat';

    if (state.actionOn >= 0 && state.seatOrder[state.actionOn] === teamID) {
        var toCall = state.currentBet - seat.bet;
        var actions = '';
        if (toCall > 0) {
            actions = '[Space/C] Call ' + toCall + '  [F] Fold';
        } else {
            actions = '[Space/C] Check';
        }
        if (seat.chips > toCall) {
            actions += '  [R] Raise ' + state.raiseAmount + '  [\u2191\u2193] Adjust';
        }
        actions += '  [A] All-in ' + seat.chips;
        return actions;
    }

    var waitName = '...';
    if (state.actionOn >= 0 && state.seats[state.seatOrder[state.actionOn]]) {
        waitName = state.seats[state.seatOrder[state.actionOn]].name;
    }
    return 'Waiting for ' + waitName + '  [Enter] Chat';
}

// ─── Helper ────────────────────────────────────────────────────────────

function getTeamForSeat(teamID) {
    var t = teams();
    for (var i = 0; i < t.length; i++) {
        if (t[i].name === teamID || teamID === t[i].name) {
            return t[i];
        }
    }
    // Look by matching any player in the team
    for (var i = 0; i < t.length; i++) {
        for (var j = 0; j < t[i].players.length; j++) {
            if (t[i].players[j].id === teamID) return t[i];
        }
    }
    return null;
}
