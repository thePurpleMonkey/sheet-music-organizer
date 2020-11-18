"use strict";

export function add_alert(title, message, style="info") {
	let alert = $("<div class='alert alert-dismissible' role='alert'>");
	alert.addClass("alert-" + style)
	alert.append($("<strong>").text(title));
	alert.append($("<div>").html(message));
	let close = $("<button type='button' class='close' aria-label='Close' data-dismiss='alert'>");
	close.append($("<span aria-hidden='true'>").html("&times;"));
	alert.append(close);

	$("#alerts").append(alert);
}


// Code snippet pulled from https://stackoverflow.com/questions/19491336/get-url-parameter-jquery-or-how-to-get-query-string-values-in-js
export function getUrlParameter(sParam) {
    var sPageURL = window.location.search.substring(1),
            sURLVariables = sPageURL.split('&'),
            sParameterName,
            i;

    for (i = 0; i < sURLVariables.length; i++) {
        sParameterName = sURLVariables[i].split('=');

        if (sParameterName[0] === sParam) {
                return sParameterName[1] === undefined ? true : decodeURIComponent(sParameterName[1]);
        }
    }
};

export function alert_ajax_failure(title, data) {
	console.log("Ajax failure!");
	console.log(data);
	if (data.status === 401) {
		let redirect = "/signin.html?redirect=" + encodeURIComponent(window.location.pathname + window.location.search);
		console.log("403 Forbidden response received. Redirecting to: " + redirect);
		window.location.href = redirect;
	}
	let alert_text = "";
	if (data.responseJSON) {
		alert_text = data.responseJSON.error;
	} else {
		alert_text = "Unknown error. Status code: " + data.status;
	}
	add_alert(title, alert_text, "danger");
}