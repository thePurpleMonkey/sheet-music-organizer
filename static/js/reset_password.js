"use strict";

import { getUrlParameter } from "./utilities.js";

// Hide options in navbar
$("#navbar_collections").hide();
$("#navbar_logout").hide();
$("#navbar_account").hide();

let token = "";
let email = "";

$(function() {
	token = getUrlParameter("token");
	console.log("Token: " + token);

	if (!token) {
		console.error("Token not provided! Redirecting to sign in page.");
		window.location.replace("signin.html");
	}
});

$("#reset").click(function() {
	let password = $("#password").val();
	let confirm = $("#confirm_password").val();

	if (password === confirm) {
		$("#wait").modal();
	} else {
		$("#reason").text("Passwords do not match!");
		$(".alert").show();
	}
});
$('#wait').on('shown.bs.modal', function (e) {
	$.post("/user/password/reset", JSON.stringify({email: email, password: $("#password").val(), token: token}))
	.done(function( data ) {
		window.location.href = "/collections.html";
	})
	.fail(function( data ) {
		if (data.responseJSON) {
			$("#reason").text(data.responseJSON.error);
		} else {
			if (data.status === 404) {
				$("#reason").text("The password reset token was not found. Please request a new password reset email.");
			} else {
				$("#reason").text("An unknown error occurred. Status code: " + data.status);
			}
		}
		$(".alert").show();
		console.log(data);
	})
	.always(function() {
		$("#wait").modal("hide");
	});
});

$("#alert-close").click(function() {
	$(".alert").hide()
});

$('#confirm_password').keypress(function (e) {
	if (e.which === 13) {
		$('#reset').click();
		return false;
	}
});