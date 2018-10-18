
layui.use(['form', 'jquery', 'element', 'layer', 'table'], function(){
    var $ = layui.$
    var HOST = "/";
    var element = layui.element;
    var form = layui.form;
    var layer = layui.layer;
    var table = layui.table;

    var recent_query = new Array();

    function addRecentQuery(q) {
        if ($.inArray(q, recent_query) >= 0) {
            return
        }
        if (recent_query.length >= 10) {
            recent_query.shift()
        }
        recent_query.push(q)
        pan = $("#recent_query_block")
        pan.html('')
        for (i = recent_query.length-1; i >=0; i--) {
            pan.append('<div class="layui-col-md1"><a href="javascript:void(0);" name="a_recent_query" style="color: #01AAED;" hash="' + recent_query[i] + '">' + recent_query[i].substr(0, 10) + '</a></div>')
        }
    }
    $(document).on("click", "a[name='a_recent_query']", function () {
        h = $(this).attr("hash")
        queryBlockDetail(h)
    })
    $(document).on("click", "a[name='block_table_hash_row']", function () {
        element.tabChange("demo", "block_detail_tab")
        h = $(this).text()
        queryBlockDetail(h)
    })
    $(document).on("click", "a[name='bonus_table_hash_row']", function () {
        h = $(this).text()
        queryBlockDetail(h)
    })
    function queryBlockDetail(h) {
        $("#query_block_hash").val(h)
        let params = {
            "method": "GTAS_blockDetail",
            "params": [h],
            "jsonrpc": "2.0",
            "id": "1"
        };
        $("#block_detail_result").hide()
        $.ajax({
            type: 'POST',
            url: HOST,
            beforeSend: function (xhr) {
                xhr.setRequestHeader("Content-Type", "application/json");
            },
            data: JSON.stringify(params),
            success: function (rdata) {
                if (rdata.result.message != "success") {
                    alert(rdata.result.message)
                    return
                }
                $("#block_detail_result").show()
                d = rdata.result.data
                $("#block__detail_height").text(d.height)
                $("#block_castor").text(d.castor)
                $("#block_hash").text(d.hash)
                $("#block_pre_hash").text(d.pre_hash)
                $("#block_ts").text(d.cur_time)
                $("#block_pre_ts").text(d.pre_time)
                $("#block_group").text(d.group_id)
                $("#block_tx_cnt").text(d.txs.length)

                gbt = d.gen_bonus_tx
                if (gbt != null && gbt != undefined) {
                    $("#gen_bonus_hash").text(gbt.hash)
                    $("#gen_bouns_value").text(gbt.value)
                    target = $("#gen_bonus_targets")
                    target.html('')
                    $.each(gbt.target_ids, function (i, v) {
                        target.append('<div class="layui-row">' + v + '</div>')
                    })
                } else {
                    $("#gen_bonus_hash").text('--')
                    $("#gen_bouns_value").text('--')
                    $("#gen_bonus_targets").html('--')
                }
                table.render({
                    elem: '#bonus_balance_table' //指定原始表格元素选择器（推荐id选择器）
                    ,cols: [[{field:'id',title: '矿工id'}, {field:'explain', title: '奖励类型'},{field:'pre_balance', title: '前块余额'}
                        ,{field:'curr_balance', title: '当前余额'},{field:'expect_balance', title: '期望余额'},{field:'expect_balance', title: '结果', templet: function (d) {
                                if (d.expect_balance == d.curr_balance) {
                                    return '<span style="color: green;">正确</span>'
                                } else {
                                    return '<span style="color: red;">错误</span>'
                                }
                            }}]] //设置表头
                    ,data: d.miner_bonus
                });
                table.render({
                    elem: '#bonus_table' //指定原始表格元素选择器（推荐id选择器）
                    ,cols: [[{field:'hash',title: 'hash'},
                        {field:'block_hash',title: '块hash', templet: '<div><a href="javascript:void(0);" class="layui-table-link" name="bonus_table_hash_row">{{d.block_hash}}</a></div>'},
                        {field:'value', title: '奖励', width:80},{field:'status_report', title: '状态', width:80},{field:'group_id', title: '组id'},
                        {field:'target_ids', title: '目标id列表'}]] //设置表头
                    ,data: d.body_bonus_txs
                });

                table.render({
                    elem: '#txs_table' //指定原始表格元素选择器（推荐id选择器）
                    ,cols: [[{field:'hash',title: 'hash', sort:true}, {field:'type', title: '类型'},{field:'source', title: '来源'}
                        ,{field:'target', title: '目标'},{field:'value', title: '金额'}]] //设置表头
                    ,data: d.trans,
                    page: true,
                    limit: 20
                });

                addRecentQuery(h)
            },
            error: function (err) {
                console.log(err)
            }
        });
    }

    $("#query_block_btn").click(function () {
        h = $("#query_block_hash").val()
        if (h == '')
            return
        queryBlockDetail(h)
    })
})