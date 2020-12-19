"use strict";

import { add_alert, alert_ajax_failure, get_session_alert, getUrlParameter } from "./utilities.js";

let collection_id = getUrlParameter("collection_id");
let create_setlist = false;

// Show options in navbar
$("#navbar_dashboard").removeClass("hidden");

// Update links
$("#collection_link").attr("href", "/collection.html?collection_id=" + collection_id);

function refreshSetlists() {
	$("#setlists_list").empty();
	$("#setlists_list").append("<li>Loading setlists, please wait...</li>");
	$.get(`/collections/${collection_id}/setlists`)
	.done(function(data) {
		console.log(data);
		$("#setlists_list").empty();

		data.forEach(setlist => {
			let a = $("<a>")
				.attr("href", "/setlist.html?collection_id=" + collection_id + "&setlist_id=" + setlist.setlist_id)
				.addClass("list-group-item list-group-item-action")
				.text(setlist.name);
			$("#setlists_list").append(a);
		});
	})
	.fail(function(data) {
		alert_ajax_failure("Unable to get setlists.", data);
	})
	.always(function() {
		$("#loading").remove();
	});
};

$(function() {
	refreshSetlists();

	// Check for any alerts
	let alert = get_session_alert();
	if (alert) {
		add_alert(alert.title, alert.message, alert.style);
	}
});

$("#create_setlist").click(function() {
	create_setlist = true;
	$("#new_setlist_modal").modal("hide");
});

$('#new_setlist_modal').on('hidden.bs.modal', function (e) {
	if (create_setlist) {
		$("#wait").modal("show");
	} else {
		$("#name").val("");
		$("#date").val("");
		$("#notes").val("");
	}
});

$('#wait').on('shown.bs.modal', function (e) {
	let payload = {
		name: $("#name").val(),
		notes: $("#notes").val(),
	};

    let date = $("#date").val();
    if (date !== "") {
        payload.date = new Date(date).toISOString();
	}
	
	$.post(`/collections/${collection_id}/setlists`, JSON.stringify(payload))
	.done(function(data) {
		console.log("Setlist post response:");
		console.log(data);
		add_alert("Setlist created!", "The setlist was successfully created.", "success");
	})
	.fail(function(data) {
		alert_ajax_failure("Unable to create setlist!", data);
	})
	.always(function() {
		create_setlist = false;
		$("#wait").modal("hide");
		refreshSetlists();
	});
});

$('#wait').on('hidden.bs.modal', function (e) {
	$("#name").val("");
	$("#date").val("");
	$("#notes").val("");
});