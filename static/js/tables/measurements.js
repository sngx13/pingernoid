$(document).ready(function () {
    var msrTable = $("#measurements").DataTable({
        "dom": '<"top"B>rt<"bottom"iflp><"clear">',
        "responsive": true,
        "lengthMenu": [10, 15, 25, 50],
        "processing": true,
        "ajax": {
            "url": "/api/v1/measurements",
            "datatype": "json",
            "contentType": "application/json; charset=utf-8",
            "cache": "false",
        },
        "columns": [
            {
                "data": null,
                render: function (data, type, row, meta) {
                    return `<a class="btn btn-xs bg-dark text-light text-start text-nowrap" href="/measurement/${data.id}"><i class="fa-solid fa-circle-info"></i> ${data.id}</a>`;
                }
            },
            { "data": "created_at" },
            { "data": "last_poll_at" },
            { "data": "target" },
            { "data": "packet_count" },
            { "data": "frequency" },
            {
                "data": "status_name",
                render: function (data, type, row, meta) {
                    let badgeClass = "";
                    switch (data) {
                        case "STOPPED":
                            badgeClass = "bg-dark";
                            break;
                        case "RUNNING":
                            badgeClass = "bg-success";
                            break;
                        case "RESTARTING":
                            badgeClass = "bg-warning";
                            break;
                        case "SCHEDULED":
                            badgeClass = "bg-info";
                            break;
                        // Add more cases as needed for other status values
                        default:
                            badgeClass = "bg-secondary";
                    }
                    return `<span class="badge ${badgeClass}">${data}</span>`;
                }
            },
            {
                "data": null,
                render: function (data, type, row, meta) {
                    if (typeof data.results === 'string' || Array.isArray(data.results)) {
                        return data.results.length;
                    } else {
                        return 0;
                    }
                }
            },
            {
                "data": null,
                render: function (data, type, row, meta) {
                    if (typeof data.alerts === 'string' || Array.isArray(data.alerts)) {
                        return data.alerts.length;
                    } else {
                        return 0;
                    }
                }
            },
            {
                "data": null,
                render: function (data, type, row, meta) {
                    return `
                    <div class="btn-group btn-group-xs" role="group">
                        <button type="button" class="btn btn-secondary text-light" hx-post="/api/v1/measurements/${data.id}/stop" hx-target="#messages" nunjucks-template="messages_template">
                            <i class="fa-solid fa-circle-stop"></i>
                        </button>
                        <button type="button" class="btn btn-success text-light" hx-post="/api/v1/measurements/${data.id}/restart" hx-target="#messages" nunjucks-template="messages_template">
                            <i class="fa-solid fa-circle-play"></i>
                        </button>
                        <button type="button" class="btn bg-danger text-light" hx-delete="/api/v1/measurements/${data.id}/delete" hx-target="#messages" nunjucks-template="messages_template">
                            <i class="fa-solid fa-trash-can"></i>
                        </button>
                    </div>`;
                }
            }
        ],
        "initComplete": function (settings, json) {
            htmx.process("#measurements");
        },
    });
    document.body.addEventListener("reloadTable", function (evt) {
        msrTable.ajax.reload(function () {
            setTimeout(htmx.process("#measurements"), 2000);
        }, false)
    });
});