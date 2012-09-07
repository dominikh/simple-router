formatByteCount = (bytes, unit = 1000, force = -1) ->
    if bytes < unit
        return bytes + " B"

    if force >= 0
        exp = force
    else
        exp = Math.floor(Math.log(bytes) / Math.log(unit))

    filler = ''
    if unit == 1000
        units = "kMGTPE"
    else
        units = "KMGTPE"
        if exp > 0
            filler = "i"


    suffix = ' ' + units.charAt(exp - 1) + filler + 'B'

    return (bytes / Math.pow(unit, exp)).toFixed(2) + suffix

byteColor = (bytes, direction) ->
    maxIn = 14107200
    maxOut = 665360

    if direction == "down"
        r = bytes / maxIn
    else
        r = bytes / maxOut


    return $.xcolor.darken([Math.floor(255 * r), Math.floor(255 * (1 - r)), 0], 2).toString()


updateStatistics = ->
    exp = $("#display_option")[0].value
    rows = $("#traffic_stats > tbody > tr")[1..-1]
    for row in rows
        for td in $(row).children()[1..-1]
            td.innerHTML = formatByteCount(td.getAttribute("data-bytes"), 1000, exp)
    return null

displayMemoryUsage = ->
    $(".system_information table.memory_usage").fadeToggle(250)

displayMain = ->
    window.active_section.fadeOut 250, ->
        window.active_section = $("#main_display")
        $("#main_display").fadeIn(250)

displayTools = ->
    window.active_section.fadeOut 250, ->
        window.active_section = $("#tools")
        $("#tools").fadeIn 250

displayNAT = ->
    table = $("#nat table#active_connections")
    table.find("tbody tr").remove()
    $("<tr><td>Loading...</td><td>Loading...</td><td>Loading...</td><td>Loading...</td></tr>").appendTo(table.find("tbody"))
    window.active_section.fadeOut 250, ->
        window.active_section = $("#nat")
        $("#nat").fadeIn(250)

    $.getJSON "/nat.json", (data) ->
        table.find("tbody tr").remove()
        for entry in data
            row = $("<tr class='" + entry.State.toLowerCase() + "'><td>" + entry.Protocol + "</td><td><a href=''>" + entry.SourceAddress  + ":" + entry.SourcePort + "</a></td><td><a href=''>" + entry.DestinationAddress + ":" + entry.DestinationPort + "</a></td><td>" + entry.State + "</td></tr>")
            row.appendTo(table.find("tbody"))
        table.trigger("update")

ellipsize = (s, length) ->
    if s.length > length
        s[0...length] + "…"
    else
        s

makeTableScroll = (el) ->
    maxRows = el.getAttribute("data-max-rows")
    wrapper = el.parentNode
    rowsInTable = el.rows.length
    height = 0
    if rowsInTable > maxRows
        for row in $(el.rows)[0...maxRows]
            height += row.clientHeight
        wrapper.style.height = height + "px"

window.uuid = null

startCapture = ->
    $.get "/uuid", "", (data) ->
        window.uuid = data
        window.location = "/traffic_capture?uuid=" + data + "&interface=" + $("#capture_interface")[0].value

stopCapture = ->
    $.get "/stop_capture", {uuid: window.uuid}

$ ->
    window.active_section = $("#main_display")

    $("#display_option").change ->
        updateStatistics()

    $("table.sortable").each (_, obj) ->
        $(obj).tablesorter()

    makeTableScroll $("#clients table")[0]

    $("#link_memory").click ->
        displayMemoryUsage()
        return false

    $("#link_nat").click ->
        displayNAT()
        return false

    $("#link_main").click ->
        displayMain()
        return false

    $("#link_tools").click ->
        displayTools()
        return false

    $("#start_capture").click ->
        $("#start_capture")[0].disabled = true
        $("#stop_capture")[0].disabled = false
        startCapture()

    $("#stop_capture").click ->
        $("#start_capture")[0].disabled = false
        $("#stop_capture")[0].disabled = true
        stopCapture()

    if location.pathname == "/"
        updateStatistics()

        jQuery.fx.interval = 50
        Highcharts.setOptions({
            global: {
                useUTC: false
            }
        })

        update = true
        backlog = []
        chart = new Highcharts.Chart({
            chart: {
                animation: {
                    duration: 400,
                    easing: 'linear',
                },
                renderTo: 'live_graph',
                type: 'areaspline',
                marginLeft: 80,
                marginRight: 10,
                showAxes: true,
                alignTicks: false,
                zoomType: "x",
            },
            title: {
                text: null
            },
            xAxis: {
                type: 'datetime',
            },
            yAxis: [
                {
                    title: false,
                    showFirstLabel: false,
                    height: 200,
                    labels: {
                        formatter: ->
                            formatByteCount(this.value, 1000) + "/s"
                    },
                },
                {
                    title: false,
                    showFirstLabel: true,
                    labels: {
                        formatter: ->
                            formatByteCount(-this.value, 1000) + "/s"
                    },
                    offset: 0,
                    top: 210,
                    height: 200,
                    max: 0,
                    threshold: 0,
                    endOnTick: true,
                    startOnTick: true,
                }
            ],
            plotOptions: {
                turboThreshold: 1,
                areaspline: {
                    fillOpacity: 0.5,
                },
            },
            tooltip: {
                crosshairs: true,
                shared: true,
            },
            legend: {
                enabled: true,
                layout: 'horizontal',
                align: 'center',
                verticalAlign: 'bottom',
                borderWidth: 0,
            },
            exporting: {
                enabled: false
            },
            credits: {
                enabled: false
            },
            series: [
                {
                    name: 'Downstream',
                    color: "#52b86f",
                    yAxis: 0,
                    data: []
                },
                {
                    name: "Upstream",
                    color: "#aa4643",
                    yAxis: 1,
                    data: [],
                    fillOpacity: 0.2
                },
            ]
        })

        $('#live_graph').hover(
            -> update = false
            -> update = true
        )
        series1 = chart.series[0]
        series2 = chart.series[1]
        if "WebSocket" of window
            socket = new WebSocket("ws://192.168.1.1:8000/traffic_data/")
        else
            socket = new MozWebSocket("ws://192.168.1.1:8000/traffic_data/")

        socket.onmessage = (msg) ->
            packet = $.parseJSON(msg.data)
            if packet.Type != "rate"
                return
            data = packet.Data
            if data.Host == "total"
                console.log(data)
                if update == true
                    for index, otherData of backlog
                        # Using the index here, because series1.data.length
                        # will not update until we call chart.redraw
                        shift = (series1.data.length + index) > 60
                        series1.addPoint([otherData.Time*1000, otherData.In], false, shift)
                        series2.addPoint([otherData.Time*1000, -otherData.Out], false, shift)
                    backlog = []

                    shift = series1.data.length > 60
                    series1.addPoint([data.Time*1000, data.In], false, shift)
                    series2.addPoint([data.Time*1000, -data.Out], false, shift)
                    chart.redraw()
                else
                    backlog.push(data)
                    backlog.shift() if backlog.length > 60
                exp = $("#display_option")[0].value
                thisMonth = $("#traffic_stats > tbody > tr:first > td")
                thisMonth[1].innerHTML = formatByteCount(data.TotalIn, 1000, exp)
                thisMonth[2].innerHTML = formatByteCount(data.TotalOut, 1000, exp)

            row = $("#clients tr[data-ip='" + data.Host + "']")[0]
            if !row
                if (data.Out == 0 && data.In == 0) || data.Host == "total"
                    return
                row = $("<tr data-ip='" + data.Host + "'><td><a href='' title='" + data.Hostname + " &lt;" + data.Host + "&gt;'>" + ellipsize(data.Hostname, 25) + "</a></td><td class='up'>↗<span class='up'>0 B/s</span></td><td class='down'>↙<span class='down'>0 B/s</span></td></tr>")
                row.appendTo("#clients tbody")

                # Adding a new row might change the graph's available
                # width, so resize the graph.
                #
                # Use the chart's container's parent to get the new
                # size.
                container = $(chart.container).parent()
                console.log(container.width(), container.height())
                chart.setSize(container.width(), container.height(), false)
            up = $(row).find("span.up")[0]
            down = $(row).find("span.down")[0]

            up.innerHTML = formatByteCount(data.Out, 1000) + "/s"
            down.innerHTML = formatByteCount(data.In, 1000) + "/s"

            $(up).css("color", byteColor(data.Out, "up"))
            $(down).css("color", byteColor(data.In, "down"))


Highcharts.Point.prototype.tooltipFormatter = (useHeader) ->
    point = this
    series = point.series
    return [
        '<span style="color:' + series.color + '">',
        (point.name || series.name),
        '</span>: ',
        '<b>',
        formatByteCount(Math.abs(point.y), 1000) + "/s",
        '</b><br />'
    ].join('')
