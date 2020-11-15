"use strict";

import { add_alert, alert_ajax_failure } from "./utilities.js";

// Hide options in navbar
$("#navbar_collections").hide();
$("#navbar_logout").hide();


$("#login").click(function() {
	$("#wait").modal();
});
$('#wait').on('shown.bs.modal', function (e) {
	$.post("/user/login", JSON.stringify({email: $("#email").val(), password: $("#password").val()}))
		.done(function( data ) {
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