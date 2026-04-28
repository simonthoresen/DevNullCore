// echo — replies to messages that start with "!echo".
// Load with: /plugin load echo

var Plugin = {
    onMessage: function(author, text, isSystem) {
        if (!isSystem && author && text.indexOf("!echo ") === 0) {
            return text.substring(6);
        }
        return null;
    }
};
