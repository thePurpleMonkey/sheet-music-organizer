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
};

export function add_session_alert(title, message, style="info") {
	try {
		window.sessionStorage.setItem("pending_invitation", true);
		window.sessionStorage.setItem("title", title);
		window.sessionStorage.setItem("message", message);
		window.sessionStorage.setItem("style", style);
		return true;
	} catch (err) {
		return false;
	}
};

export function get_session_alert() {
	if (window.sessionStorage.getItem("pending_invitation") === "true") {
		try {
			window.sessionStorage.setItem("pending_invitation", false);
			return {
				title: window.sessionStorage.getItem("title"),
				message: window.sessionStorage.getItem("message"),
				style: window.sessionStorage.getItem("style"),
			};
		} catch (err) {
			return null;
		}
	} else {
		return null;
	}
};

export function substitute_URLs(text) {
	let urls = [];
	// var urlRegex = /(https?:\/\/[^\s]+)/g;
	let urlRegex = /(https?:\/\/)?[\w\-~]+(\.[\w\-~]+)+(\/[\w\-~@:%]*)*(#[\w\-]*)?(\?[^\s]*)?/gi;
	let result = text.replace(urlRegex, function(match) {
		let url = match;
		let suffix = "";
		let prefix = "";
		if (match.endsWith('?')) {
			url = url.substring(0, url.length - 1);
			suffix = '?';
		}
		if (!match.startsWith("http")) {
			prefix = "//";
		}
		urls.push(url);
		return '<a href="' + prefix + url + '" target="_blank" rel="noreferrer noopener">' + url + '</a>' + suffix;
	});

	return {html: result, URLs: urls};
  };

export function get_youtube_video_id(url){
    var regExp = /^.*((youtu.be\/)|(v\/)|(\/u\/\w\/)|(embed\/)|(watch\?))\??v?=?([^#&?]*).*/;
    var match = url.match(regExp);
    return (match&&match[7].length==11)? match[7] : false;
}