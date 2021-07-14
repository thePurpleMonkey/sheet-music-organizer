"use strict";

import { add_alert, alert_ajax_failure } from "./utilities.js";

// Hide options in navbar
$("#navbar_collections").hide();
$("#navbar_logout").hide();
$("#navbar_account").hide();

$("#register").click(function() {
	$("#wait").modal();
});
$("#wait").on('shown.bs.modal', function (e) {
	if ($("#password").val() !== $("#confirm").val()) {
		add_alert("Passwords don't match!", "The passwords don't match. Please re-enter your password.", "danger");
		$("#confirm").val("");
		$("#wait").modal("hide");
		return;
	}
	$.post( "/user/register", JSON.stringify({ email: $("#email").val(), password: $("#password").val(), name: $("#name").val() }) )
		.done(function( data ) {
			// Set the user to be logged in
			try {
				window.localStorage.setItem("logged_in", true);
			} catch (err) {
				console.log("Unable to set localStorage variable 'logged_in' to true.");
				console.log(err);
			}
			
			// Save the user_id
			try {
				console.log("Login response data:");
				console.log(data);
				window.localStorage.setItem("user_id", data.user_id);
			} catch (err) {
				console.log("Unable to set localStorage variable 'user_id'");
				console.log(err);
			}

			// Redirect to next URL
			let redirect = new URL(window.location.href).searchParams.get("redirect");
			if (redirect === null) {
				redirect = "/collections.html";
			}
			redirect = decodeURIComponent(redirect);
			console.log("Redirecting to: " + redirect);
			window.location.href = redirect;
		})
		.fail(function( data ) {
			alert_ajax_failure("Registration failed.", data);
			console.log(data);
		})
		.always(function() {
			$("#wait").modal("hide");
		});
});

$('#name').keypress(function (e) {
	if (e.which === 13) {
		$('#register').click();
		return false;
	}
});