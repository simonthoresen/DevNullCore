// greeter — auto-welcomes new players when they join.
// Load with: /plugin load greeter

var Plugin = {
    onMessage: function(author, text, isSystem) {
        // System messages about players joining look like "PlayerName joined."
        if (isSystem && text.match(/^(.+) joined\.$/)) {
            var name = text.match(/^(.+) joined\.$/)[1];
            return "Welcome, " + name + "! Type /help to get started.";
        }
        return null;
    }
};
