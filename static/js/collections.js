"use strict";

import { add_alert, alert_ajax_failure, create_alert, get_session_alert } from "./utilities.js";

let create_collection = false;
let pending_invitations = [];

function refreshCollections() {
	$("#collections").empty();
	$("#collections").append("<li>Loading collections, please wait...</li>");
	$.get("/collections")
	.done(function(data) {
		console.log(data);
		$("#collections").empty();

		data.forEach(collection => {
			let a = $("<a>")
				.attr("href", "/collection.html?collection_id=" + collection.collection_id)
				.addClass("list-group-item list-group-item-action")
				.text(collection.name);
			$("#collections").append(a);
		});
		
		if (data.length == 0) {
			show_tutorial();
		}
	})
	.fail(function(data) {
		if (data.status == 403) {
			console.log("Not authorized to access collections. Redirecting to sign in page...");
			window.location.replace("/signin.html");
		}

		alert_ajax_failure("Unable to get collections.", data);
	})
	.always(function() {
		$("#loading").remove();
	});
};

$(function() {
	// Enable form validation
	$("#create_collection_form").validate();

	// Load collections
	refreshCollections();

	// Check for any alerts
	let alert = get_session_alert();
	if (alert) {
		add_alert(alert.title, alert.message, alert.style);
	}

	// Check for any pending invitations for this user
	check_pending_invitations();
});

// #region Create collection
$("#create_collection").click(function() {
	if ($("#create_collection_form").valid()) {
		create_collection = true;
		$("#new_collection_modal").modal("hide");
	}
});

$('#new_collection_modal').on('hidden.bs.modal', function (e) {
	if (create_collection) {
		$("#wait").modal("show");
	}
});

$('#wait').on('shown.bs.modal', function (e) {
	let payload = JSON.stringify({name: $("#name").val(), description: $("#description").val()});
	$.post("/collections", payload)
	.done(function(data) {
		console.log(data);
		add_alert("Collection created!", "The collection was successfully created.", "success");

		// Show next step of tutorial
		$("#new_collection_tutorial_alert").hide();
		$("#open_collection_tutorial_alert").show();
	})
	.fail(function(data) {
		alert_ajax_failure("Unable to create collection!", data);
	})
	.always(function() {
		create_collection = false;
		$("#wait").modal("hide");
		refreshCollections();
	});
});
// #endregion

// #region Pending invitations
function check_pending_invitations() {
	$.get("/user/invitations")
	.done(function(data) {
		console.log("User invitations response:");
		console.log(data);

		pending_invitations = data;

		if (data.length > 0) {
			let plural = data.length > 1;
			add_alert(`${data.length} pending invitation${plural ? "s" : ""}`, `You have ${plural ? " " : "a "}pending invitation${plural ? "s" : ""}. <a href="javascript:;" class="alert-link" data-toggle="modal" data-target="#pending_invitations_modal">View pending invitations</a>.`, "info");
		}
	})
	.fail(function(data) {
		console.warn("Unable to get user's pending invitations.");
		console.log(data);
	});
}

$("#pending_invitations_modal").on("show.bs.modal", function() {
	pending_invitations.forEach(function(invitation) {
		let element = $("<a class='list-group-item list-group-item-action'>")
		.attr("href", "/accept_invite.html?token=" + invitation.token)
		.text(invitation.collection_name)

		$("#pending_invitations_list").append(element);
	});
});

// #endregion

// #region Tutorial
function hide_tutorial() {
	console.log("Hide tutorial clicked.");
	try {
		window.localStorage.setItem("show_tutorial", false);
	} catch (err) {
		console.warn("Unable to hide tutorial");
		console.warn(err);
	} finally {
		$(".tutorial").hide(500);
	}
}

function show_tutorial() {
	let tutorial = window.localStorage.getItem("show_tutorial");
	console.log("Tutorial: " + tutorial);
	if (tutorial != "false") {
		$("#new_collection_tutorial_alert").removeClass("hidden");
		$("#create_collection_tutorial_alert").removeClass("hidden");
		$(".hide_tutorial").click(hide_tutorial);
	}
}
// #endregion