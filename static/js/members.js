"use strict";

import { add_alert, alert_ajax_failure, getUrlParameter } from "./utilities.js";

let collection_id = getUrlParameter("collection_id");

function refresh_invitations() {
	$("#invitations_list").empty();
	$("#invitations_list").append("<li>Loading pending invitations, please wait...</li>");
	$.get(`/collections/${collection_id}/invitations`)
	.done(function(data) {
		console.log(data);
		$("#invitations_list").empty();

		if (data.length === 0) {
			let item = $("<li>").addClass("list-group-item disabled").attr("aria-disabled", "true").text("No pending invitations");
			$("#invitations_list").append(item);
		}

		data.forEach(invite => {
			let item = $("<li>")
				.addClass("list-group-item");
			item.append(invite.email);
			$("#invitations_list").append(item);
		});
	})
	.fail(function(data) {
		alert_ajax_failure("Unable to get pending invitations.", data);
		$("#invitations_list").empty();
		$("#invitations_list").append($("<li>").addClass("list-group-item disabled").attr("aria-disabled", "true").text("Error retrieving pending invitations."));
	});
}

$(function() {
    // Replace link for collection
	$("#collection_link").attr("href", "/collection.html?collection_id=" + collection_id);
	
	// Populate list of collection members
	$("#members_list").empty();
	$("#members_list").append("<li>Loading members, please wait...</li>");
	$.get(`/collections/${collection_id}/members`)
	.done(function(data) {
		console.log(data);
		$("#members_list").empty();

		data.forEach(member => {
			let item = $("<li>")
				.addClass("list-group-item");
			if (member.admin)
			{
				item.append(
					$("<img>")
					.attr("src", "/img/key.svg")
					.addClass("admin-icon")
					.attr("title", "Administrator")
					);
			}
			item.append(member.name);
			$("#members_list").append(item);
		});
	})
	.fail(function(data) {
		alert_ajax_failure("Unable to get members.", data);
	});

	// Get collection name
	$.get(`/collections/${collection_id}`)
	.done(function(data) {
		console.log("GET collection info:")
		console.log(data);
		$("#collection_name").text(data.name);
	})
	.fail(function(data) {
		alert_ajax_failure("Unable to get collection name.", data);
	});

	// Refresh invitations
	refresh_invitations();
});

// Logout button
$("#logout").click(function() {
	$.get("/user/logout");
	window.location.href = "/"
});