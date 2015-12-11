(function() {

    var load = $(document).on("pageload", update())


    $(document).ready(function(){
      var r_ct=1;
      var c_ct=1;
      $("#add_row").click(function(){
        $('#tab_logic').append("<tr class='dt-row' id='dt-row"+r_ct+"' r='"+r_ct+"'><td class='dt-col0'><input class='form-control' r='"+r_ct+"' c='0' type='text'></td></tr>");

      // $('#tab_logic').append('<tr class="dt-row" id="dt-row'+(r_ct+1)+'" r="'+(r_ct+1)+'"></tr>');
      r_ct++; 
     });
     $("#delete_row").click(function(){
         if(r_ct>1){
         $("#dt-row"+(r_ct-1)).remove();
         r_ct--;
         }
     });

     $("#add_col").click(function(){
        for (i=0;i<r_ct;i++){
            $("#dt-row"+i).append("<td class='dt-col"+c_ct+"'><input class='form-control' r='"+$("#dt-row"+i).attr("r")+"' c='"+c_ct+"' type='text'></td>")
        }
        // $(".dt_row").each(function(){
        //     var row = $(this)
        //     row.append("<td><input class='form-control dt-col' id='dt-col"
        //         +c_ct+"' r='"+row.attr("r")+"' c='"+c_ct+"' type='text'></td>")
        // })
        c_ct++;
     })

     $("#delete_col").click(function(){
        if(c_ct>1){
            $(".dt-col"+(c_ct-1)).each(function(){
                $(".dt-col"+(c_ct-1)).remove()
            })

            c_ct--
        }
     })

    });

    $("#bu1").click(function(event) {

        $.get('/api/data', function(data) {
            var j = JSON.parse(data)
            var cellInput = j.Data
            $(".ginput").each(function() {
                var cell = $(this)

                cell.val(cellInput[cell.attr("grow")][cell.attr("gcol")])
            })
        })

    })

    var timer = setInterval(update, 2000)

    function update() {
        $.get('/api/data', function(data) {
            var j = JSON.parse(data)
            var cellInput = j.Data
            $(".ginput").each(function() {
                var cell = $(this)
                if (!cell.is(":focus")) {
                    cell.val(cellInput[cell.attr("grow")][cell.attr("gcol")])
                }
            })
        })
    }

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