// Scanlines shader — dims alternating rows to simulate a CRT scanline effect.
// The scanlines scroll slowly downward over time.
const Shader = {
    process(buf, time) {
        var start = Math.floor((time * 3) % 4) % 2;
        for (var y = start; y < buf.height; y += 2) {
            buf.recolor(0, y, buf.width, 1, null, null, ATTR_FAINT);
        }
    }
};
