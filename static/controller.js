(function() {
    if (!String.format) {
        String.format = function(format) {
            var args = Array.prototype.slice.call(arguments, 1);
            return format.replace(/{(\d+)}/g, function(match, number) {
                return typeof args[number] != 'undefined' ? args[number] : match;
            });
        };
    }

    window.gg = {}
    window.gg.oldVal = ""

    $(".ginput").each(function() {
        var cell = $(this)

        cell.focus(function(event) {
            window.gg.oldVal = cell.val()
            console.log(String.format("focus on row:{0},col:{1}", cell.attr("grow"), cell.attr("gcol")));
        });

        cell.blur(function(event) {
            if (window.gg.oldVal === cell.val()) {
                console.log("unchanged")
            } else {
                console.log("changed")
            }
            console.log(String.format("blur on row:{0},col:{1}", cell.attr("grow"), cell.attr("gcol")));
        })
    })

})()