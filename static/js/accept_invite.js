"use strict";

import { add_alert, alert_ajax_failure, getUrlParameter } from "./utilities.js";

let collection_id = getUrlParameter("collection_id");
let token = getUrlParameter("token");

$(function() {
	let payload = {token: token};
	let redirect_to = encodeURIComponent(window.location.href);

	// Set add redirect to links
	$(".needs_token").attr("href", function(i, href) {
		return href + "?redirect=" + redirect_to;
	});

	// Get invitation
	$.get(`/invitations`, payload)
	.done(function(data) {
		console.log(data);
		// Do something
		$("#email").text(data.inviter_email);
		$("#name").text(data.inviter_name);		
		$("#collection").text(data.collection_name);		
		$("#admin").text(data.administrator ? "Yes" : "No");		
		$("#confirm_invite").show(500);
	})
	.fail(function(data) {
		console.log("Error getting invitation:");
		console.log(data);
		if (data.status === 403) {
			if (data.responseJSON.code == "wrong_user") {
				add_alert("Wrong user", "You cannot accept this invitation. Please log out and try again.", "danger");
				$("#logout").show();
			} else if (data.responseJSON.code == "retracted") {
				add_alert("Invitation retracted.", data.responseJSON.error, "danger");
			} else {
				add_alert("Unknown error", data.responseJSON, "danger");
			}
		} else if (data.status === 401) {
			$("#unauthorized").show(500);
			return;
		} else {
			alert_ajax_failure("Unable to get invitation.", data);
		}
		$("#failed").show(500);
	})
	.always(function() {
		$("#loading").hide(500);
	});
});

$("#join").click(function() {
	$("#join_wait").modal("show");
});

$("#join_wait").on("shown.bs.modal", function() {
	let payload = JSON.stringify({token: token});
	$.post(`/invitations`, payload)
	.done(function(data) {
		console.log("Accept invitation response:")
		console.log(data);
		add_alert("You have been added to the collection!", "You should be redirected to the collection momentarily.", "success");
		window.location.href = `/collections.html?collection_id=${collection_id}`
	})
	.fail(function(data) {
		alert_ajax_failure("Unable to get invitation.", data);
	})
	.always(function(data) {
		$("#join_wait").hide(500);
	})
});

// Logout button
$("#logout").click(function() {
	$.get("/user/logout");
	window.location.reload();
});