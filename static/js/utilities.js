"use strict";

export function create_alert(title, message, style, id=undefined, custom_class=undefined) {
	let alert = $("<div class='alert alert-dismissible' role='alert'>");
	alert.addClass("alert-" + style)
	alert.append($("<strong>").text(title));
	alert.append($("<div>").html(message).addClass("alert_message"));
	let close = $("<button type='button' class='close' aria-label='Close' data-dismiss='alert'>");
	close.append($("<span aria-hidden='true'>").html("&times;"));
	alert.append(close);

	if (id) {
		alert.attr("id", id);
	}

	if (custom_class) {
		alert.addClass(custom_class);
	}

	return alert;
}

export function add_alert(title, message, style="info", id=undefined, custom_class=undefined) {
	$("#alerts").append(create_alert(title, message, style, id, custom_class));
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
		console.log("403 Forbidden response received. Setting 'logged_in' to false.");
		try { window.localStorage.setItem("logged_in", false); } catch(err) { console.log("Unable to 'logged_in' localStorage variable to false."); console.log(err); }
		console.log("Redirecting to: " + redirect);
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

export function is_user_logged_in(default_value = false) {
	try {
		return window.localStorage.getItem("logged_in") === "true";
	} catch (err) {
		console.error(`Error checking user login status. Returning default value ${default_value}`);
		console.error(err);
		return default_value;
	}
}

export function substitute_URLs(text) {
	let urls = [];
	// var urlRegex = /(https?:\/\/[^\s]+)/g;
	let urlRegex = /(https?:\/\/)?[\w\-~]+(\.[\w\-~]+)+(\/[\w\-~@:%\!\.]*)*(#[\w\-]*)?(\?[^\s]*)?/gi;
	let result = text.replace(urlRegex, function(match) {
		let url = match;
		let suffix = "";
		let prefix = "";
		if (match.endsWith('?') || match.endsWith('!')) {
			suffix = url.substr(url.length - 1, 1); // Get the last character of the url
			url = url.substring(0, url.length - 1); // Trim the last character of the url
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
};

// Source: http://stackoverflow.com/questions/497790
export var dates = {
    convert:function(d) {
        // Converts the date in d to a date-object. The input can be:
        //   a date object: returned without modification
        //  an array      : Interpreted as [year,month,day]. NOTE: month is 0-11.
        //   a number     : Interpreted as number of milliseconds
        //                  since 1 Jan 1970 (a timestamp) 
        //   a string     : Any format supported by the javascript engine, like
        //                  "YYYY/MM/DD", "MM/DD/YYYY", "Jan 31 2009" etc.
        //  an object     : Interpreted as an object with year, month and date
        //                  attributes.  **NOTE** month is 0-11.
        return (
            d.constructor === Date ? d :
            d.constructor === Array ? new Date(d[0],d[1],d[2]) :
            d.constructor === Number ? new Date(d) :
            d.constructor === String ? new Date(d) :
            typeof d === "object" ? new Date(d.year,d.month,d.date) :
            NaN
        );
    },
    compare:function(a,b) {
        // Compare two dates (could be of any type supported by the convert
        // function above) and returns:
        //  -1 : if a < b
        //   0 : if a = b
        //   1 : if a > b
        // NaN : if a or b is an illegal date
        // NOTE: The code inside isFinite does an assignment (=).
        return (
            isFinite(a=this.convert(a).valueOf()) &&
            isFinite(b=this.convert(b).valueOf()) ?
            (a>b)-(a<b) :
            NaN
        );
    },
    inRange:function(d,start,end) {
        // Checks if date in d is between dates in start and end.
        // Returns a boolean or NaN:
        //    true  : if d is between start and end (inclusive)
        //    false : if d is before start or after end
        //    NaN   : if one or more of the dates is illegal.
        // NOTE: The code inside isFinite does an assignment (=).
       return (
            isFinite(d=this.convert(d).valueOf()) &&
            isFinite(start=this.convert(start).valueOf()) &&
            isFinite(end=this.convert(end).valueOf()) ?
            start <= d && d <= end :
            NaN
        );
    }
};

// #region Tutorial

export function disable_tutorial(user_id, page=undefined) {
	let key;
	if (page === undefined) {
		key = `show_tutorial_user_${user_id}`;
	} else {
		key = `show_tutorial_user_${user_id}_${page}`
	}

	console.log(`Disabling tutorial key: ${key}`);
	try {
		window.localStorage.setItem(key, false);
	} catch (err) {
		console.warn("Unable to disable tutorial");
		console.warn(err);
	}
}

export function is_tutorial_enabled(user_id, page=undefined) {
	let key;
	if (page === undefined) {
		key = `show_tutorial_user_${user_id}`;
	} else {
		key = `show_tutorial_user_${user_id}_${page}`
	}
	console.log("Loading local storage key: " + key);
	let tutorial = window.localStorage.getItem(key);
	console.log("Tutorial: " + tutorial);
	return tutorial != "false";
}
// #endregion
