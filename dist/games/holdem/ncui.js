// ncui.js — NC-style box drawing library for null-space JS games
// Renders bordered panels and layouts as plain text using the same
// box-drawing characters as the framework chrome.

// ─── Border Characters (matching server/theme.go defaults) ─────────────
var NC = {
    // Outer border (double-line, NC windows)
    TL: '\u2554', TR: '\u2557', BL: '\u255A', BR: '\u255D',
    H:  '\u2550', V:  '\u2551',
    // Inner dividers (single-line, NC panels)
    IH: '\u2500', IV: '\u2502',
    // Intersections (inner meets outer)
    XL: '\u255F', XR: '\u2562', XT: '\u2564', XB: '\u2567', XX: '\u253C',
    // Single-line box (for inner panels)
    STL: '\u250C', STR: '\u2510', SBL: '\u2514', SBR: '\u2518'
};

// ─── String Helpers ────────────────────────────────────────────────────
function ncSpaces(n) {
    var s = '';
    for (var i = 0; i < n; i++) s += ' ';
    return s;
}

function ncRepeat(ch, n) {
    var s = '';
    for (var i = 0; i < n; i++) s += ch;
    return s;
}

// Strip ANSI escape codes for measuring visible length
function ncStripAnsi(str) {
    return str.replace(/\x1b\[[0-9;]*m/g, '');
}

function ncVisLen(str) {
    return ncStripAnsi(str).length;
}

// Pad or truncate a string to exactly `w` visible characters, preserving ANSI
function ncFit(str, w) {
    var vis = ncStripAnsi(str);
    if (vis.length === w) return str;
    if (vis.length > w) {
        // Truncate: walk through original string tracking visible chars
        var out = '';
        var seen = 0;
        var inEsc = false;
        for (var i = 0; i < str.length && seen < w; i++) {
            var ch = str[i];
            if (ch === '\x1b') { inEsc = true; out += ch; continue; }
            if (inEsc) { out += ch; if (ch === 'm') inEsc = false; continue; }
            out += ch;
            seen++;
        }
        return out;
    }
    // Pad with spaces
    return str + ncSpaces(w - vis.length);
}

// Center text within width
function ncCenter(str, w) {
    var vis = ncVisLen(str);
    if (vis >= w) return ncFit(str, w);
    var left = Math.floor((w - vis) / 2);
    return ncSpaces(left) + str + ncSpaces(w - vis - left);
}

// Right-align text within width
function ncRight(str, w) {
    var vis = ncVisLen(str);
    if (vis >= w) return ncFit(str, w);
    return ncSpaces(w - vis) + str;
}

// ─── Panel Rendering ───────────────────────────────────────────────────

// Render a bordered panel with optional title.
// Returns an array of strings, each exactly `width` chars wide.
// `lines` is an array of content strings (will be padded/truncated).
// Options: { title, double (default true), align: 'left'|'center'|'right' }
function ncPanel(width, height, lines, opts) {
    if (!opts) opts = {};
    var title = opts.title || '';
    var dbl = opts.double !== false; // default to double border
    var tl = dbl ? NC.TL : NC.STL;
    var tr = dbl ? NC.TR : NC.STR;
    var bl = dbl ? NC.BL : NC.SBL;
    var br = dbl ? NC.BR : NC.SBR;
    var h  = dbl ? NC.H  : NC.IH;
    var v  = dbl ? NC.V  : NC.IV;

    var innerW = width - 2; // subtract left+right border
    var result = [];

    // Top border with optional title
    var topBar = ncRepeat(h, innerW);
    if (title && title.length > 0) {
        var t = ' ' + title + ' ';
        if (t.length > innerW) t = t.substring(0, innerW);
        var tpad = Math.floor((innerW - t.length) / 2);
        topBar = ncRepeat(h, tpad) + t + ncRepeat(h, innerW - tpad - t.length);
    }
    result.push(tl + topBar + tr);

    // Content rows
    var contentH = height - 2; // subtract top+bottom border
    for (var y = 0; y < contentH; y++) {
        var line = (lines && y < lines.length) ? lines[y] : '';
        result.push(v + ncFit(line, innerW) + v);
    }

    // Bottom border
    result.push(bl + ncRepeat(h, innerW) + br);

    return result;
}

// Render a panel with a horizontal divider after the title row.
// titleContent is rendered in the first row, then a divider, then body lines.
function ncPanelWithHeader(width, height, titleContent, bodyLines, opts) {
    if (!opts) opts = {};
    var title = opts.title || '';
    var innerW = width - 2;
    var result = [];

    // Top border
    var topBar = ncRepeat(NC.H, innerW);
    if (title) {
        var t = ' ' + title + ' ';
        if (t.length > innerW) t = t.substring(0, innerW);
        var tpad = Math.floor((innerW - t.length) / 2);
        topBar = ncRepeat(NC.H, tpad) + t + ncRepeat(NC.H, innerW - tpad - t.length);
    }
    result.push(NC.TL + topBar + NC.TR);

    // Title content row
    result.push(NC.V + ncFit(titleContent || '', innerW) + NC.V);

    // Horizontal divider
    result.push(NC.XL + ncRepeat(NC.IH, innerW) + NC.XR);

    // Body rows
    var bodyH = height - 3; // top border + title + divider + bottom border = 4 overhead, but we have 3 already
    bodyH = height - result.length - 1; // subtract what we have + bottom border
    for (var y = 0; y < bodyH; y++) {
        var line = (bodyLines && y < bodyLines.length) ? bodyLines[y] : '';
        result.push(NC.V + ncFit(line, innerW) + NC.V);
    }

    // Bottom border
    result.push(NC.BL + ncRepeat(NC.H, innerW) + NC.BR);

    return result;
}

// ─── Layout: Side-by-Side Panels ───────────────────────────────────────

// Merge two arrays of equal-length strings side by side.
function ncHConcat(leftLines, rightLines) {
    var maxLen = Math.max(leftLines.length, rightLines.length);
    var leftW = leftLines.length > 0 ? ncVisLen(leftLines[0]) : 0;
    var rightW = rightLines.length > 0 ? ncVisLen(rightLines[0]) : 0;
    var result = [];
    for (var y = 0; y < maxLen; y++) {
        var l = y < leftLines.length ? leftLines[y] : ncSpaces(leftW);
        var r = y < rightLines.length ? rightLines[y] : ncSpaces(rightW);
        result.push(l + r);
    }
    return result;
}

// Merge three arrays side by side.
function ncHConcat3(a, b, c) {
    return ncHConcat(ncHConcat(a, b), c);
}

// ─── Table Rendering ───────────────────────────────────────────────────

// Render rows as a simple aligned table within a given width.
// rows: array of arrays of strings (columns).
// colWidths: array of column widths (visible chars). If null, auto-calculated.
function ncTable(rows, colWidths, width) {
    if (!rows || rows.length === 0) return [];
    var cols = rows[0].length;

    if (!colWidths) {
        colWidths = [];
        for (var c = 0; c < cols; c++) {
            var maxW = 0;
            for (var r = 0; r < rows.length; r++) {
                if (rows[r][c]) {
                    var w = ncVisLen(rows[r][c]);
                    if (w > maxW) maxW = w;
                }
            }
            colWidths.push(maxW);
        }
    }

    var result = [];
    for (var r = 0; r < rows.length; r++) {
        var line = '';
        for (var c = 0; c < cols; c++) {
            var cell = rows[r][c] || '';
            line += ncFit(cell, colWidths[c]);
            if (c < cols - 1) line += ' ';
        }
        result.push(line);
    }
    return result;
}

// ─── Button Rendering ──────────────────────────────────────────────────

// Render a button like [ Label ]
function ncButton(label, focused) {
    if (focused) {
        return '\x1b[7m[ ' + label + ' ]\x1b[0m';
    }
    return '[ ' + label + ' ]';
}

// ─── Progress Bar ──────────────────────────────────────────────────────

function ncProgressBar(width, fraction, filledChar, emptyChar) {
    if (!filledChar) filledChar = '\u2588';
    if (!emptyChar) emptyChar = '\u2591';
    var filled = Math.floor(width * Math.max(0, Math.min(1, fraction)));
    return ncRepeat(filledChar, filled) + ncRepeat(emptyChar, width - filled);
}

// ─── Full Screen Layout ────────────────────────────────────────────────

// Fill a lines array to exactly `height` lines, each `width` chars.
function ncFillScreen(lines, width, height) {
    var result = [];
    for (var y = 0; y < height; y++) {
        if (y < lines.length) {
            result.push(ncFit(lines[y], width));
        } else {
            result.push(ncSpaces(width));
        }
    }
    return result;
}
