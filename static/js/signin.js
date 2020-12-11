"use strict";

import { add_alert, alert_ajax_failure } from "./utilities.js";

// Hide options in navbar
$("#navbar_collections").hide();
$("#navbar_logout").hide();
$("#navbar_account").hide();


$("#login").click(function() {
	$("#wait").modal();
});
$('#wait').on('shown.bs.modal', function (e) {
	let payload = {
		email: $("#email").val(),
		password: $("#password").val(),
		remember: $("#remember_me").prop("checked"),
	};
	console.log("Payload:");
	console.log(payload);
	$.post("/user/login", JSON.stringify(payload))
		.done(function( data ) {
			// Set the user to be logged in
			try {
				window.localStorage.setItem("logged_in", true);
			} catch (err) {
				console.log("Unable to set localStorage variable 'logged_in' to true.");
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
			if (data.status === 401) {
				add_alert("Sign in failed.", "Username or password incorrect.", "danger");
			} else {
				alert_ajax_failure("Sign in failed.", data);
			}
			console.log(data)
		})
		.always(function() {
			console.log("Hiding modal...")
			$("#wait").modal("hide")
		});
});

$('#password').keypress(function (e) {
	if (e.which === 13) {
		$('#login').click();
		return false;
	}
});

$('#remember_me').keypress(function (e) {
	if (e.which === 13) {
		$('#login').click();
		return false;
	}
});