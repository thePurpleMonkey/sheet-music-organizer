"use strict";

import { add_alert, alert_ajax_failure } from "./utilities.js";

$("#submit_message").click(function() {
    $("#submit_message").prop("disabled", true);
    $("#wait").removeClass("hidden");

    let payload = {
        name: $("#name").val(),
        email: $("#email").val(),
        message: $("#message").val(),
    }

    console.log(payload);

    $.post("/contact", JSON.stringify(payload))
    .done(function(data) {
        console.log("Contact response:");
        console.log(data);

        add_alert("Message sent", "Your message has been sent.", "success")
        $("#name").val("");
        $("#email").val("");
        $("#message").val("");
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to send message", data);
    })
    .always(function() {
        $("#submit_message").prop("disabled", false);
        $("#wait").addClass("hidden");
    });
});
