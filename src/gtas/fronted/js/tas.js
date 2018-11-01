
layui.use(['form', 'jquery', 'element', 'layer', 'table'], function(){
    var element = layui.element;
    var form = layui.form;
    var layer = layui.layer;
    var $ = layui.$;
    var HOST = "/";
    var ref;
    var host_ele = $("#host");
    var online=false;
    var current_block_height = -1;
    host_ele.text(HOST);
    var blocks = [];
    var groups = [];
    var lastReloadBlockSize = 0
    var lastReloadGroupSize = 0
    var workGroups = [];
    var groupIds = new Set();
    var table = layui.table;
    var dashboard_update_switch = true


    var block_table = table.render({
        elem: '#block_detail' //指定原始表格元素选择器（推荐id选择器）
        ,initSort: {
            field: 'height',
            type: 'asc'
        }
        ,cols: [[{field:'height',title: '块高', sort:true},
            {field:'hash', title: 'hash', templet: '<div><a href="javascript:void(0);" class="layui-table-link" name="block_table_hash_row">{{d.hash}}</a></div>'},
            {field:'pre_hash', title: 'pre_hash'},{field:'pre_time', title: 'pre_time', width: 189},{field:'cur_time', title: 'cur_time', width: 189},
            {field:'castor', title: 'castor'},{field:'group_id', title: 'group_id'}, {field:'txs', title: 'tx_count'}, {field:'qn', title: 'qn'}
            , {field:'total_qn', title: 'totalQN'}]] //设置表头
        ,data: blocks
        ,page: true
        ,limit:15
    });

    var group_table = table.render({
        elem: '#group_detail' //指定原始表格元素选择器（推荐id选择器）
        ,initSort: {
            field: 'begin_height',
            type: 'asc'
        }
        ,cols: [[{field:'height',title: '高度', sort: true, width:140}, {field:'group_id',title: '组id', width:140}, {field:'dummy', title: 'dummy', width:80},
            {field:'parent', title: '父亲组', width:140},{field:'pre', title: '上一组', width:140},
            {field:'begin_height', title: '生效高度', width: 100},{field:'dismiss_height', title: '解散高度', width:100},
            {field:'members', title: '成员列表'}]] //设置表头
        ,data: groups
        ,page: true
        ,limit:15
    });

    var work_group_table = table.render({
        elem: '#work_group_detail' //指定原始表格元素选择器（推荐id选择器）
        ,cols: [[{field:'id',title: '组id', width:140}, {field:'parent', title: '父亲组', width:140},{field:'pre', title: '上一组', width:140},
            {field:'begin_height', title: '生效高度', width: 100},{field:'dismiss_height', title: '解散高度', width:100},
            {field:'group_members', title: '成员列表'}]] //设置表头
        ,data: groups
        ,page: true
        ,limit:15
    });

    $("#dashboard_update_div").click(function () {
        console.log('dashboard_update_switch click')
        if ($("#dashboard_update_switch").is(':checked')){
            dashboard_update_switch = true
            updateDashboardUpdate()
        } else {
            dashboard_update_switch = false
            updateDashboardUpdate()
        }
    });

    $("#change_host").click(function () {
        layer.prompt({
            formType: 0,
            value: HOST,
            title: '请输入新的host',
        }, function(value, index, elem){
            HOST = value;
            host_ele.text(HOST);
            layer.close(index);
            current_block_height = 0;
            blocks = [];
            block_table.reload({
                data: blocks
            });
        });
    });

    // 查询余额
    $(".query_btn").click(function () {
        let id = $(this).attr("id");
        let count = id.split("_")[2];
        $("#balance_message").text("");
        $("#balance_error").text("");
        let params = {
            "method": "GTAS_balance",
            "params": [$("#query_input_"+count).val()],
            "jsonrpc": "2.0",
            "id": "1"
        };
        $.ajax({
            type: 'POST',
            url: HOST,
            beforeSend: function (xhr) {
                xhr.setRequestHeader("Content-Type", "application/json");
            },
            data: JSON.stringify(params),
            success: function (rdata) {
                if (rdata.result !== undefined){
                    $("#balance_message").text(rdata.result.message);
                    $("#query_balance_"+count).text(rdata.result.data)
                }
                if (rdata.error !== undefined){
                    $("#balance_error").text(rdata.error.message);
                }
            },
        });
    });

    // 钱包初始化
    function init_wallets() {
        let params = {
            "method": "GTAS_getWallets",
            "params": [],
            "jsonrpc": "2.0",
            "id": "1"
        };
        $.ajax({
            type: 'POST',
            url: HOST,
            beforeSend: function (xhr) {
                xhr.setRequestHeader("Content-Type", "application/json");
            },
            data: JSON.stringify(params),
            success: function (rdata) {
                if (rdata.result !== undefined){
                    $.each(rdata.result.data, function (i,val) {
                        let tr = "<tr class='wallet_tr'><td>" + val.private_key + "</td><td>" + val.address
                            + '</td><td ><button class="layui-btn wallet_del">删除</button></td></tr>';
                        $("#create_chart").append(tr);

                        $(".wallet_del").click(function () {
                            let parent = $(this).parents("tr");
                            del_wallet(parent.children("td:eq(1)").text());
                            parent.remove();
                        });
                    })
                }
            },
        });
    }

    function del_wallet(key) {
        let params = {
            "method": "GTAS_deleteWallet",
            "params": [key],
            "jsonrpc": "2.0",
            "id": "1"
        };
        $.ajax({
            type: 'POST',
            url: HOST,
            beforeSend: function (xhr) {
                xhr.setRequestHeader("Content-Type", "application/json");
            },
            data: JSON.stringify(params),
            success: function (rdata) {

            },
        });
    }

    init_wallets();


    // 创建钱包
    $("#create_btn").click(function () {
        let params = {
            "method": "GTAS_newWallet",
            "params": [],
            "jsonrpc": "2.0",
            "id": "1"
        };
        $.ajax({
            type: 'POST',
            url: HOST,
            beforeSend: function (xhr) {
                xhr.setRequestHeader("Content-Type", "application/json");
            },
            data: JSON.stringify(params),
            success: function (rdata) {
                let tr = "<tr class='wallet_tr'><td>" + rdata.result.data.private_key + "</td><td>" + rdata.result.data.address
                    + '</td><td ><button class="layui-btn wallet_del">删除</button></td></tr>';
                $("#create_chart").append(tr);

                $(".wallet_del").click(function () {
                    let parent = $(this).parents("tr");
                    del_wallet(parent.children("td:eq(1)").text());
                    parent.remove();
                });
            },
        });
    });

    // 投票表单提交
    form.on('submit(vote_form)', function (data) {
        $("#vote_message").text("");
        $("#vote_error").text("");
        let from = data.field.from;
        let vote_param = {};
        vote_param.template_name = data.field.template_name;
        vote_param.p_index = parseInt(data.field.p_index);
        vote_param.p_value = data.field.p_value;
        vote_param.custom = (data.field.custom === "true");
        vote_param.desc = data.field.desc;
        vote_param.deposit_min = parseInt(data.field.deposit_min);
        vote_param.total_deposit_min = parseInt(data.field.total_deposit_min);
        vote_param.voter_cnt_min = parseInt(data.field.voter_cnt_min);
        vote_param.approval_deposit_min = parseInt(data.field.approval_deposit_min);
        vote_param.approval_voter_cnt_min = parseInt(data.field.approval_voter_cnt_min);
        vote_param.deadline_block = parseInt(data.field.deadline_block);
        vote_param.stat_block = parseInt(data.field.stat_block);
        vote_param.effect_block = parseInt(data.field.effect_block);
        vote_param.deposit_gap = parseInt(data.field.deposit_gap);
        let params = {
            "method": "GTAS_vote",
            "params": [from, vote_param],
            "jsonrpc": "2.0",
            "id": "1"
        };
        $.ajax({
            type: 'POST',
            url: HOST,
            beforeSend: function (xhr) {
                xhr.setRequestHeader("Content-Type", "application/json");
            },
            data: JSON.stringify(params),
            success: function (rdata) {
                if (rdata.result !== undefined){
                    $("#vote_message").text(rdata.result.message)
                }
                if (rdata.error !== undefined){
                    $("#vote_error").text(rdata.error.message)
                }
            },
        });
        return false;
    });


    // 交易表单提交
    form.on('submit(t_form)', function(data){
        $("#t_message").text("");
        $("#t_error").text("");
        let from = data.field.from;
        let to = data.field.to;
        let amount = data.field.amount;
        let code = data.field.code;
        let t = data.field.type;
        let nonce = data.field.nonce;
        // if (from.length !== 42) {
        //     layer.msg("from 参数字段长度错误");
        //     return false
        // }
        // if (to.length !== 42) {
        //     layer.msg("to 参数字段长度错误");
        //     return false
        // }
        let params = {
            "method": "GTAS_t",
            "params": [from, to, parseFloat(amount), code, parseInt(nonce), parseInt(t)],
            "jsonrpc": "2.0",
            "id": "1"
        };

        $.ajax({
            type: 'POST',
            url: HOST,
            beforeSend: function (xhr) {
                xhr.setRequestHeader("Content-Type", "application/json");
            },
            data: JSON.stringify(params),
            success: function (rdata) {
                if (rdata.result !== undefined){
                    $("#t_message").text(rdata.result.message)
                }
                if (rdata.error !== undefined){
                    $("#t_error").text(rdata.error.message)
                }
            },
        });
        return false;
    });

    // 同步块信息
    function syncBlock(height) {
        let params = {
            "method": "GTAS_getBlock",
            "params": [height],
            "jsonrpc": "2.0",
            "id": "1"
        };
        $.ajax({
            type: 'POST',
            url: HOST,
            beforeSend: function (xhr) {
                xhr.setRequestHeader("Content-Type", "application/json");
            },
            data: JSON.stringify(params),
            success: function (rdata) {
                if (rdata.result !== undefined){
                    blocks.push(rdata.result.data);
                    if (current_block_height < rdata.result.data) {
                        current_block_height = rdata.result.data
                    }
                    // block_table.reload({
                    //         data: blocks
                    //     }
                    // )
                }
                if (rdata.error !== undefined){
                    // $("#t_error").text(rdata.error.message)
                }
            },
        });
    }

    // 同步组信息
    function syncGroup(height) {
        let params = {
            "method": "GTAS_getGroupsAfter",
            "params": [height],
            "jsonrpc": "2.0",
            "id": "1"
        };
        $.ajax({
            type: 'POST',
            url: HOST,
            beforeSend: function (xhr) {
                xhr.setRequestHeader("Content-Type", "application/json");
            },
            data: JSON.stringify(params),
            success: function (rdata) {
                if (rdata.result !== undefined && rdata.result != null && rdata.result.message == 'success'){
                    retArr = rdata.result.data
                    for(i = 0; i < retArr.length; i++) {
                        if (!groupIds.has(retArr[i]["group_id"])) {
                            groups.push(retArr[i])
                            groupIds.add(retArr[i]["group_id"])
                        }
                    }
                    // group_table.reload({
                    //         data: groups
                    //     }
                    // )
                }
                if (rdata.error !== undefined){
                    // $("#t_error").text(rdata.error.message)
                }
            },
            error: function (err) {
                console.log(err)
            }
        });
    }

    $("#query_wg_btn").click(function () {
        var h = $("#query_wg_input").val()
        if (h == null || h == undefined || h == '') {
            alert("请输入查询高度")
            return
        }
        queryWorkGroup(parseInt(h))
    })
    //查询工作组
    function queryWorkGroup(height) {
        let params = {
            "method": "GTAS_getWorkGroup",
            "params": [height],
            "jsonrpc": "2.0",
            "id": "1"
        };
        $.ajax({
            type: 'POST',
            url: HOST,
            beforeSend: function (xhr) {
                xhr.setRequestHeader("Content-Type", "application/json");
            },
            data: JSON.stringify(params),
            success: function (rdata) {
                if (rdata.result !== undefined && rdata.result != null && rdata.result.message == 'success'){
                    retArr = rdata.result.data
                    work_group_table.reload({
                            data: retArr
                        }
                    )
                }
                if (rdata.error !== undefined){
                    // $("#t_error").text(rdata.error.message)
                }
            },
            error: function (err) {
                console.log(err)
            }
        });
    }

    $(document).on("click", "a[name='miner_oper_a']", function () {
        m = $(this).attr("method")
        t = $(this).attr("mtype")
        let params = {
            "method": m,
            "params": [parseInt(t)],
            "jsonrpc": "2.0",
            "id": "1"
        };
        text = $(this).text()
        $.ajax({
            type: 'POST',
            url: HOST,
            beforeSend: function (xhr) {
                xhr.setRequestHeader("Content-Type", "application/json");
            },
            data: JSON.stringify(params),
            success: function (rdata) {
                if (rdata.result.message == "success") {
                    $(this).text("已申请"+text)
                } else {
                    alert(rdata.result.message)
                }
            },
            error: function (err) {
                console.log(err)
            }
        });
    })

    $("#apply_a").click(function () {
        f = $("#apply_miner_div")
        if(f.is(":hidden")){
            $(this).text("取消申请")
            f.show()
        }else{
            $(this).text("申请成为矿工")
            f.hide()
        }

    })

    $("#submit_apply").click(function () {
        stake = parseInt($("#text_stake").val())
        t = parseInt($("input[name='app_type_rd']:checked").val())
        $("#submit_result").text("")
        let params = {
            "method": "GTAS_minerApply",
            "params": [stake, t],
            "jsonrpc": "2.0",
            "id": "1"
        };
        $.ajax({
            type: 'POST',
            url: HOST,
            beforeSend: function (xhr) {
                xhr.setRequestHeader("Content-Type", "application/json");
            },
            data: JSON.stringify(params),
            success: function (rdata) {
                $("#submit_result").text(rdata.result.message)
            },
            error: function (err) {
                console.log(err)
            }
        });
    })

    function reloadBlocksTable() {
        if (blocks.length == lastReloadBlockSize)
            return
        block_table.reload({
            data: blocks
        })
        lastReloadBlockSize = blocks.length
    }
    function reloadGroupsTable() {
        if (groups.length == lastReloadGroupSize)
            return
        group_table.reload({
            data: groups
        })
        lastReloadGroupSize = groups.length
    }

    function dashboardLoad() {
        let params = {
            "method": "GTAS_dashboard",
            "params": [],
            "jsonrpc": "2.0",
            "id": "1"
        };
        $.ajax({
            type: 'POST',
            url: HOST,
            beforeSend: function (xhr) {
                xhr.setRequestHeader("Content-Type", "application/json");
            },
            data: JSON.stringify(params),
            success: function (rdata) {
                d = rdata.result.data
                //块高
                $("#block_height").text(d.block_height)
                clear = $("#tb_node_status").text() == "已停止" && d.node_info.status == "运行中"
                if (clear) {
                    current_block_height = 0;
                    blocks = [];
                    block_table.reload({
                        data: blocks
                    })
                }

                for(let i=current_block_height+1; i<=d.block_height; i++) {
                    syncBlock(i);
                }
                current_block_height = d.block_height

                //组高
                $("#group_height").text(d.group_height)
                if (clear) {
                    groups = []
                    groupIds.clear()
                    group_table.reload({
                        data: groups
                    })
                } else {
                    if (groups.length == 0) {
                        syncGroup(0)
                    } else if (groups[groups.length-1]["height"]+1 < d.group_height) {
                        syncGroup(groups[groups.length-1]["height"]+1)
                    }
                }
                //工作组数量
                $("#work_group_num").text(d.work_g_num)
                if ($("#tb_node_id").text() != d.node_info.id) {
                    //节点和质押信息
                    $("#tb_node_id").text(d.node_info.id)
                    $("#tx_send_from").val(d.node_info.id)
                }
                $("#tb_node_balance").text(d.node_info.balance)
                $("#tb_node_status").text(d.node_info.status)
                $("#tb_node_type").text(d.node_info.n_type)
                $("#tb_node_wg").text(d.node_info.w_group_num)
                $("#tb_node_ag").text(d.node_info.a_group_num)
                $("#tb_node_txnum").text(d.node_info.tx_pool_num)
                $("#tb_stake_body").html("")
                $.each(d.node_info.mort_gages, function (i, v) {
                    tr = "<tr>"
                    tr += "<td>" + v.stake + "</td>"
                    tr += "<td>" + v.type + "</td>"
                    tr += "<td>" + v.apply_height + "</td>"
                    tr += "<td>" + v.abort_height + "</td>"
                    mtype = 0
                    if (v.type == "重节点") {
                        mtype = 1
                    }
                    if (v.abort_height > 0) {
                        tr += "<td><a href=\"javascript:void(0);\" name='miner_oper_a' method='GTAS_minerRefund' mtype=" + mtype + ">退款</a></td>"
                    } else {
                        tr += "<td><a href=\"javascript:void(0);\" name='miner_oper_a' method='GTAS_minerAbort' mtype=" + mtype + ">取消</a></td>"
                    }
                    tr += "</tr>"
                    $("#tb_stake_body").append(tr)
                })

                //链接节点
                let nodes_table = $("#nodes_table");
                nodes_table.empty();
                d.conns.sort(function (a, b) {
                    return parseInt(a.ip.split(".")[3]) - parseInt(b.ip.split(".")[3])
                });
                $.each(d.conns, function (i,val) {
                    nodes_table.append(
                        " <tr><td>id</td><td>ip</td><td>port</td></tr>".replace("ip", val.ip).replace("id", val.id).replace("port", val.tcp_port)
                    )
                })
            },
            error: function (err) {
                console.log(err)
                $("#tb_node_status").text("已停止")
                $("#trans_table").empty();
                $("#nodes_table").empty();
            }
        });
    }

    // dashboard同步数据
    // syncNodes();
    // syncTrans();
    // syncBlockHeight();
    // syncGroupHeight();
    syncGroup(0)
    dashboardLoad()

    dashboard = setInterval(function(){
        dashboardLoad();
    },1000);

    function updateDashboardUpdate(){
        if (dashboard_update_switch){
            dashboard = setInterval(function(){
                dashboardLoad();
            },1000);
        } else{
            clearInterval(dashboard)
        }
    }

    blocktable_inter = 0
    grouptable_inter = 0

    element.on('tab(demo)', function(data){
        // if(data.index === 0) {
        //     clearInterval(dashboard)
        //     dashboard = setInterval(function(){
        //         dashboardLoad()
        //     },1000);
        // } else {
        //     clearInterval(dashboard)
        // }

        if (data.index == 5) {
            reloadBlocksTable()
            blocktable_inter = setInterval(GGTreloadBlocksTable, 5000);
        } else {
            clearInterval(blocktable_inter)
        }
        if (data.index == 6) {
           reloadGroupsTable()
            grouptable_inter = setInterval(reloadGroupsTable, 5000);
        } else {
            clearInterval(grouptable_inter)
        }

    });

});