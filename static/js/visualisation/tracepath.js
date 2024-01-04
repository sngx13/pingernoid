function drawTracePath(msrID) {
    var url = "/api/v1/measurements/" + msrID + "/traceroute/path";
    $.getJSON(url, function (response) {
        var container = document.getElementById("tracePath");
        var data = {
            nodes: response.data.nodes,
            edges: response.data.edges
        };
        var options = {
            manipulation: false,
            autoResize: true,
            height: "100%",
            width: "100%",
            locale: "en",
            layout: {
                improvedLayout: true,
                hierarchical: {
                    enabled: false,
                }
            },
            edges: {
                arrows: {
                    to: {
                        enabled: true,
                        type: "arrow",
                    }
                },
                color: {
                    color: "#eb1313",
                    highlight: "#35c90c",
                    hover: "#35c90c"
                }
            },
            nodes: {
                shape: "box",
                size: 16,
                color: {
                    border: "#000000",
                    background: "#2b2e2b",
                    highlight: {
                        border: "#35c90c",
                        background: "#2b2e2b"
                    },
                    hover: {
                        border: "#35c90c",
                        background: "#2b2e2b"
                    }
                },
                font: {
                    size: 10,
                    face: "arial",
                    color: "#ffffff"
                },
            },
            physics: {
                enabled: true,
                forceAtlas2Based: {
                    theta: 0.5,
                    gravitationalConstant: -50,
                    centralGravity: 0.01,
                    springConstant: 0.08,
                    springLength: 100,
                    damping: 0.4,
                    avoidOverlap: 0
                },
                stabilization: {
                    enabled: true,
                    iterations: 1000,
                    updateInterval: 100,
                    onlyDynamicEdges: false,
                    fit: true
                },
                solver: "forceAtlas2Based",
            }
        };
        var network = new vis.Network(container, data, options);
        network.on("stabilized", function (params) {
            network.fit({ animation: { duration: 1000, easingFunction: "easeInOutQuad" } });
        });
    });
};