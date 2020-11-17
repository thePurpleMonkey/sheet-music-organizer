"use strict";

import { add_alert, alert_ajax_failure, getUrlParameter } from "./utilities.js";

let send_verification_email = false;

// Show navbar options
$("#navbar_options").show();

// Enable tooltips:
$("#help").tooltip();

$(function() {
	$.get("/user/account")
	.done(function(data) {
		console.log("Account info:");
		console.log(data);

		$("#email").text(data.email);
		$("#name").text(data.name);
		if (data.verified) {
			$("#verified").show();
		} else {
			$("#not_verified").show();
		}
		$("#name").text(data.name);
	})
	.fail()
	.always();
});

$("#send_verification_email").click(function() {
	send_verification_email = true;
	$("#verify_confirm_modal").modal("hide");
});
$("#verify_confirm_modal").on("hidden.bs.modal", function() {
	if (send_verification_email) {
		$("#verify_wait").modal("show");
	};
	send_verification_email = false;
})
$("#verify_wait").on("shown.bs.modal", function() {
	$.post("/user/verify")
	.done(function(data) {
		add_alert("Verification email sent", "Your account verification email has been sent. Please allow up to 15 minutes for the email to arrive in your inbox. Check your spam messages if the email is not in your inbox.")
	})
	.fail(function(data) {
		alert_ajax_failure("Unable to send verification email.", data);
	})
	.always(function() {
		$("#verify_wait").modal("hide");
	})
})