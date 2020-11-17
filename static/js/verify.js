"use strict";

import { add_alert, alert_ajax_failure, getUrlParameter } from "./utilities.js";

let token = getUrlParameter("token");

$(function() {
	let payload = {token: token};

	// Get invitation
	$.get(`/user/verify`, payload)
	.done(function(data) {
		console.log("Verify GET response:");
		console.log(data);		
		$("#success").show(500);
		window.sessionStorage.setItem("verified", true);
		window.location.href = "/collections.html";
	})
	.fail(function(data) {
		console.log("Error verifying account:");
		console.log(data);
		alert_ajax_failure("Unable to verify account.", data);
		$("#failed").show(500);
	})
	.always(function() {
		$("#loading").hide(500);
	});
});
