// Rainbow shader — cycles border characters through a shifting rainbow palette.
const Shader = {
    process(buf, time) {
        var t = time;
        for (var y = 0; y < buf.height; y++) {
            for (var x = 0; x < buf.width; x++) {
                var p = buf.getPixel(x, y);
                if (!p) continue;
                if (!isBorder(p.char)) continue;
                // Hue shifts based on position + time for a flowing rainbow.
                var hue = ((x + y) * 12 + t * 120) % 360;
                var rgb = hslToHex(hue, 0.9, 0.55);
                buf.setChar(x, y, p.char, rgb, p.bg, p.attr);
            }
        }
    }
};

function isBorder(ch) {
    // Box drawing characters are in the range U+2500–U+257F,
    // plus the double-line and rounded variants.
    var code = ch.charCodeAt(0);
    return code >= 0x2500 && code <= 0x257F;
}

function hslToHex(h, s, l) {
    h = h / 360;
    var r, g, b;
    if (s === 0) {
        r = g = b = l;
    } else {
        var q = l < 0.5 ? l * (1 + s) : l + s - l * s;
        var p = 2 * l - q;
        r = hue2rgb(p, q, h + 1/3);
        g = hue2rgb(p, q, h);
        b = hue2rgb(p, q, h - 1/3);
    }
    return "#" + toHex(Math.round(r * 255)) + toHex(Math.round(g * 255)) + toHex(Math.round(b * 255));
}

function hue2rgb(p, q, t) {
    if (t < 0) t += 1;
    if (t > 1) t -= 1;
    if (t < 1/6) return p + (q - p) * 6 * t;
    if (t < 1/2) return q;
    if (t < 2/3) return p + (q - p) * (2/3 - t) * 6;
    return p;
}

function toHex(n) {
    var hex = n.toString(16);
    return hex.length < 2 ? "0" + hex : hex;
}
