// Scanlines shader — dims alternating rows to simulate a CRT scanline effect.
// The scanlines scroll slowly downward over time.
const Shader = {
    offset: 0,

    update(dt) {
        this.offset = (this.offset + dt * 3) % 4;
    },

    process(buf) {
        var start = Math.floor(this.offset) % 2;
        for (var y = start; y < buf.height; y += 2) {
            buf.recolor(0, y, buf.width, 1, null, null, ATTR_FAINT);
        }
    }
};
