"use strict";

import { add_alert, alert_ajax_failure, getUrlParameter } from "./utilities.js";

let send_verification_email = false;
let save_edit = false;

// Show navbar options
$("#navbar_options").removeClass("hidden");

// Enable tooltips:
$("#help").tooltip();
$("#edit_email").tooltip({
	title: "Editing your email will require you to verify your account again.",
	trigger: "focus"
});

function reset_ui() {
	$("#email").text("Loading...");
	$("#name").text("Loading...");
	$("#verified").hide();
	$("#not_verified").hide();
	$("#verify_loading").show();
}

function refresh_account() {
	$.get("/user/account")
	.done(function(data) {
		console.log("Account info:");
		console.log(data);

		$("#email").text(data.email);
		$("#edit_email").val(data.email);
		$("#name").text(data.name);
		$("#edit_name").val(data.name);
		$("#verify_loading").hide();
		if (data.verified) {
			$("#verified").show();
		} else {
			$("#not_verified").show();
		}
		$("#name").text(data.name);
	})
	.fail(function(data) {
		alert_ajax_failure("Unable to retrieve account info.", data);
	});
}

$(function() {
	refresh_account();
});

// Verification
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
});

// Edit Account
$("#edit_button").click(function() {
	$("#edit_modal").modal("show");
});
$("#save_button").click(function() {
	save_edit = true;
	$("#edit_modal").modal("hide");
});
$("#edit_modal").on("hidden.bs.modal", function() {
	if (save_edit) {
		$("#edit_wait").modal("show");
	}
});
$("#edit_wait").on("shown.bs.modal", function() {
	let payload = JSON.stringify({
		email: $("#edit_email").val(),
		name: $("#edit_name").val(),
	});
	$.ajax("/user/account", {
		method: "PUT",
		data: payload
	})
	.done(function(data) {
		console.log("Update account info response:");
		console.log(data);
		reset_ui();
		refresh_account();
		add_alert("Account info updated.", "Your account has been successfully updated.", "success");
	})
	.fail(function(data) {
		alert_ajax_failure("Unable to update account.", data);
	})
	.always(function() {
		$("#edit_wait").modal("hide");
	});
});