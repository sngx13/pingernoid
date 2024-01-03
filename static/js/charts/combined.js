function drawCharts(msrID, timeRange) {
    // Rtt Chart Options
    var rttChartOptions = {
        theme: {
            mode: "light",
            palette: "palette1",
        },
        chart: {
            animations: {
                enabled: false
            },
            type: "line",
            height: 260,
            width: "100%",
            zoom: {
                autoScaleYaxis: true,
                enabled: true
            },
            toolbar: {
                autoSelected: "zoom"
            }
        },
        markers: {
            size: 0
        },
        dataLabels: {
            enabled: false
        },
        stroke: {
            width: 2,
            curve: "straight"
        },
        xaxis: {
            type: "datetime"
        },
        yaxis: {
            type: "numeric",
            decimalsInFloat: 0,
            title: { text: "Milliseconds" },
            labels: {
                formatter: function (value) {
                    return value.toFixed(0) + "ms";
                }
            },
            style: {
                fontSize: "8px",
                fontFamily: "Roboto"
            }
        },
        tooltip: {
            shared: true,
            intersect: false,
            x: {
                format: "dd/MM/yy HH:mm"
            }
        },
        legend: {
            show: true,
            style: {
                fontSize: "8px",
                fontFamily: "Roboto"
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
    // Pkt Chart Options
    var pktChartOptions = {
        theme: {
            mode: "light",
            palette: "palette1"
        },
        chart: {
            animations: {
                enabled: false
            },
            stacked: true,
            stackType: "50%",
            type: "bar",
            height: 260,
            width: "100%",
            zoom: {
                autoScaleYaxis: true,
                enabled: true
            },
            toolbar: {
                autoSelected: "zoom"
            }
        },
        markers: {
            size: 0
        },
        dataLabels: {
            enabled: false,
        },
        xaxis: {
            type: "datetime",
        },
        yaxis: {
            type: "numeric",
            title: { text: "Packet Count" },
            style: {
                fontSize: "8px",
                fontFamily: "Roboto"
            }
        },
        tooltip: {
            shared: true,
            intersect: false,
            x: {
                format: "dd/MM/yy HH:mm"
            }
        },
        legend: {
            show: true,
            style: {
                fontSize: "8px",
                fontFamily: "Roboto"
            }
        },
        noData: {
            text: "Loading..."
        },
        responsive: [
            {
                breakpoint: 1000,
                options: {
                    plotOptions: {
                        bar: {
                            horizontal: false
                        }
                    },
                    legend: {
                        position: "bottom"
                    }
                }
            }
        ],
        series: [],
    };
    // Hop Chart Options
    var hopChartOptions = {
        theme: {
            mode: "light",
            palette: "palette1"
        },
        chart: {
            animations: {
                enabled: false
            },
            type: "bar",
            height: 260,
            width: "100%",
            zoom: {
                autoScaleYaxis: true,
                enabled: true
            },
            toolbar: {
                autoSelected: "zoom"
            }
        },
        markers: {
            size: 0
        },
        dataLabels: {
            enabled: false,
        },
        xaxis: {
            type: "datetime",
        },
        yaxis: {
            type: "numeric",
            title: { text: "Hop Count" },
            style: {
                fontSize: "8px",
                fontFamily: "Roboto"
            }
        },
        tooltip: {
            shared: true,
            intersect: false,
            x: {
                format: "dd/MM/yy HH:mm"
            }
        },
        legend: {
            show: true,
            style: {
                fontSize: "8px",
                fontFamily: "Roboto"
            }
        },
        noData: {
            text: "Loading..."
        },
        responsive: [
            {
                breakpoint: 1000,
                options: {
                    plotOptions: {
                        bar: {
                            horizontal: false
                        }
                    },
                    legend: {
                        position: "bottom"
                    }
                }
            }
        ],
        series: [],
    };
    // Rtt Chart
    var msrRttChart = new ApexCharts(
        document.querySelector("#measurement_rtt_chart"),
        rttChartOptions
    );
    msrRttChart.render();
    // Pkt Chart
    var msrPktChart = new ApexCharts(
        document.querySelector("#measurement_pkt_chart"),
        pktChartOptions
    );
    msrPktChart.render();
    // Hop Chart
    var msrHopChart = new ApexCharts(
        document.querySelector("#measurement_hop_chart"),
        hopChartOptions
    );
    msrHopChart.render();
    // Get chart data
    var url = "/api/v1/measurements/" + msrID + "/results/combined/" + timeRange;
    $.getJSON(url, function (response) {
        // Latency Statistics
        var jitterData = response.data.Rtt.Jitter;
        var latencyMinData = response.data.Rtt.LatencyMin;
        var latencyMaxData = response.data.Rtt.LatencyMax;
        var latencyAvgData = response.data.Rtt.LatencyAvg;
        msrRttChart.updateSeries([jitterData, latencyMinData, latencyMaxData, latencyAvgData]);
        // Packet Statistics
        var lostPacketsData = response.data.Pkt.PacketsLost;
        var sentPacketsData = response.data.Pkt.PacketsSent;
        var rcvdPacketsData = response.data.Pkt.PacketsRcvd;
        msrPktChart.updateSeries([lostPacketsData, sentPacketsData, rcvdPacketsData]);
        // Hop Count Statistics
        var ipHopCountData = response.data.Hop.IPHopCount;
        var asHopCountData = response.data.Hop.ASHopCount;
        msrHopChart.updateSeries([ipHopCountData, asHopCountData]);
    });
};