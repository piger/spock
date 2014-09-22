CodeMirror.defineMode("wikiEditor", function(config) {
    var mode;
    if (pageMode !== "") {
        mode = pageMode;
    } else {
        mode = "markdown";
    }

    return CodeMirror.multiplexingMode(
        CodeMirror.getMode(config, mode),
        {open: /^---$/, close: /^---$/,
         mode: CodeMirror.getMode(config, "yaml"),
         delimStyle: "delimit"}
    );
});

var editor = CodeMirror.fromTextArea(document.getElementById("content-area"), {
    mode: "wikiEditor",
    lineNumbers: true,
    theme: "mdn-like",
    styleActiveLine: true
});

$(document).ready(function() {
    $("#cancel-btn").on('click', function () {
        var u = window.location.href;
        window.location.href = u.replace("?action=edit", "");
    });
});
