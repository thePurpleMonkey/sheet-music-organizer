"use strict";

import { add_alert, alert_ajax_failure } from "/js/utilities.js";

// Show navbar options
$("#navbar_logout").hide();
$("#navbar_account").hide();
$("#navbar_login").show();
$("#navbar_register").show();

$("#request_reset").click(function() {
	$("#wait").modal("show");
});
$('#wait').on('shown.bs.modal', function (e) {
	$.post("/user/password/forgot", JSON.stringify({email: $("#email").val()}))
		.done(function( data ) {
			$("form").hide(500);
			$("#email_sent").show(500);
		})
		.fail(function(data) {
			console.log(data);
			alert_ajax_failure(data);
		})
		.always(function() {
			$("#wait").modal("hide")
		});
});

$('#email').keypress(function (e) {
	if (e.which === 13) {
		$('#request_reset').click();
		return false;
	}
});