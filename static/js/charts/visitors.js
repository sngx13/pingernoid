function drawVisitorsChart() {
    var visitorsChartOptions = {
        theme: {
            mode: "light",
            palette: "palette1",
        },
        chart: {
            animations: {
                enabled: false
            },
            type: "treemap",
            height: 480,
            width: "100%",
        },
        dataLabels: {
            enabled: true,
            style: {
                fontSize: "12px",
                fontFamily: "Roboto",
            },
            formatter: function (text, op) {
                return [text, "Visitors: " + op.value]
            },
            offsetY: -4
        },
        plotOptions: {
            treemap: {
                distributed: true,
                enableShades: false
            }
        },
        noData: {
            text: "Loading..."
        },
        responsive: [
            {
                breakpoint: 1000,
                options: {
                    legend: {
                        position: "bottom"
                    }
                }
            }
        ],
        series: [],
    };
    // Pkt Chart
    var visitorsChart = new ApexCharts(
        document.querySelector("#visitors_chart"),
        visitorsChartOptions
    );
    visitorsChart.render();
    // Get chart data
    var url = "/api/v1/site/visitor/info/chart";
    $.getJSON(url, function (response) {
        var responseData = response.data;
        var modifiedData = responseData.map(entry => {
            return { x: entry.X, y: entry.Y };
        });
        visitorsChart.updateSeries([{
            name: "Count",
            data: modifiedData
        }])
    });
};