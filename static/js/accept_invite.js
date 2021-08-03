"use strict";

import { add_alert, add_session_alert, alert_ajax_failure, getUrlParameter } from "./utilities.js";

var collection_id;
let token = getUrlParameter("token");
let recipient_email = getUrlParameter("email");

// Hide options in navbar
$("#navbar_collections").hide();

$(function() {
	let payload = {token: token};
	let redirect_to = encodeURIComponent(window.location.href);

	// Set add redirect to links
	$(".needs_token").attr("href", function(i, href) {
		return href + "?redirect=" + redirect_to
					+ "&email=" + recipient_email;
	});

	// Get invitation
	$.get(`/invitations`, payload)
	.done(function(data) {
		console.log(data);

		// Load invitation data
		$("#email").text(data.inviter_email);
		$("#name").text(data.inviter_name);		
		$("#collection").text(data.collection_name);		
		$("#admin").text(data.administrator ? "Yes" : "No");
		collection_id = data.collection_id;

		// Add email to login and register links
		$(".needs_token").attr("href", function(i, href) {
			return href + "&email=" + data.email;
		});

		// Show invite
		$("#confirm_invite").removeClass("hidden");
	})
	.fail(function(data) {
		console.log("Error getting invitation:");
		console.log(data);
		if (data.status === 403) {
			if (data.responseJSON.code == "wrong_user") {
				add_alert("Wrong user", "You cannot accept this invitation. Please log out and try again.", "danger");
				$("#logout").removeClass(hidden);
			} else if (data.responseJSON.code == "retracted") {
				add_alert("Invitation retracted.", data.responseJSON.error, "danger");
			} else {
				add_alert("Unknown error", data.responseJSON, "danger");
			}
		} else if (data.status === 401) {
			$("#unauthorized").removeClass("hidden");
			return;
		} else {
			alert_ajax_failure("Unable to get invitation.", data);
		}
		$("#failed").removeClass("hidden");
	})
	.always(function() {
		$("#loading").addClass("hidden");
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
		if (add_session_alert("You have been added to the collection!", "You can now contribute to this collection.", "success")) {
			window.location.href = `/collection.html?collection_id=${collection_id}`
		}
	})
	.fail(function(data) {
		alert_ajax_failure("Unable to get invitation.", data);
	})
	.always(function(data) {
		$("#join_wait").addClass("hidden");
	})
});
