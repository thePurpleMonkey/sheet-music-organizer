"use strict";

import { add_alert, alert_ajax_failure, getUrlParameter } from "./utilities.js";

let collection_id = getUrlParameter("collection_id");
let send_invite = false;

let datetime_format = new Intl.DateTimeFormat([], {
	dateStyle: "short",
	timeStyle: "short"
})

// Hide options in navbar
$("#navbar_dashboard").show();

function refresh_invitations() {
	$("#invitations_list").empty();
	$("#invitations_list").append("<li>Loading pending invitations, please wait...</li>");
	$.get(`/collections/${collection_id}/invitations`)
	.done(function(data) {
		console.log("Invitations list:");
		console.log(data);
		$("#invitations_list").empty();

		if (data.length === 0) {
			let item = $("<li>").addClass("list-group-item disabled").attr("aria-disabled", "true").text("No pending invitations");
			$("#invitations_list").append(item);
		}

		data.forEach(invite => {
			let invite_sent = Date.parse(invite.invite_sent)
			let item = $("<li>")
				.addClass("list-group-item")
				.attr("title", "Invite sent at " + datetime_format.format(invite_sent));
			item.append(invite.invitee_email);
			
			// Delete button
			let retract_invite_button = $("<button type='button' class='close' title='Retract invitation'>");
			retract_invite_button.append($("<span aria-hidden='true'>").html("&times;"));
			
			// Store invitation_id with this element
			retract_invite_button.data("invitation_id", invite.invitation_id);

			retract_invite_button.click(retract_invite);
			item.append(retract_invite_button);

			$("#invitations_list").append(item);
		});
	})
	.fail(function(data) {
		alert_ajax_failure("Unable to get pending invitations.", data);
		$("#invitations_list").empty();
		$("#invitations_list").append($("<li>").addClass("list-group-item disabled").attr("aria-disabled", "true").text("Error retrieving pending invitations."));
	});
};

function retract_invite(e) {
	let invitation_id = $(this).data("invitation_id");
	console.log("Retracting invitation_id: " + invitation_id);
	if (!invitation_id) {
		add_alert("Unable to retract invitation!", "This operation has failed.", "danger");
		return;
	}

	$.ajax(`/collections/${collection_id}/invitations/${invitation_id}`, {
		method: "DELETE"
	})
	.done(function(data) {
		console.log("Retract invitation response:")
		console.log(data);
	})
	.fail(function(data) {
		alert_ajax_failure("Unable to retract invitation.", data);
	})
	.always(function(data) {
		refresh_invitations();
		add_alert("Success!", "Successfully retracted invitation.", "success");
	});
};

function refresh_members() {
	$("#members_list").empty();
	$("#members_list").append("<li>Loading members, please wait...</li>");
	$.get(`/collections/${collection_id}/members`)
	.done(function(data) {
		console.log("Members list:");
		console.log(data);
		$("#members_list").empty();

		let user_id = data.user_id;
		let admin = data.admin;

		if (admin) {
			$("#invite_button").show();
		} else {
			$("#invite_button").hide();
		}

		data.members.forEach(member => {
			let item = $("<li>")
				.addClass("list-group-item");
			if (member.admin)
			{
				item.append(
					$("<img>")
					.attr("src", "/img/key.svg")
					.addClass("admin-icon")
					.attr("title", "Administrator")
					.attr("alt", "Administrator")
					);
			}
			item.append(member.name);

			if (user_id === member.user_id) {
				item.append(
					$("<img>")
					.attr("src", "/img/user.svg")
					.attr("alt", "Your account")
					.attr("title", "Your account")
					.addClass("user_img")
				);
			} else if (admin) {
				let remove_member_button = $("<button type='button' class='close' title='Remove member from collection'>");
				remove_member_button.append($("<span aria-hidden='true'>").html("&times;"));
				
				// Store user_id with this element
				remove_member_button.data("user_id", member.user_id);

				remove_member_button.click(remove_member);
				item.append(remove_member_button);
			}

			$("#members_list").append(item);
		});
	})
	.fail(function(data) {
		alert_ajax_failure("Unable to get members.", data);
	});
}

function remove_member(e) {
	let user_id = $(this).data("user_id");
	console.log("Removing user_id: " + user_id);
	if (!user_id) {
		add_alert("Unable to remove user", "This operation has failed.", "danger");
		return;
	}

	$.ajax(`/collections/${collection_id}/members/${user_id}`, {
		method: "DELETE"
	})
	.done(function(data) {
		console.log("Delete member response:")
		console.log(data);
	})
	.fail(function(data) {
		alert_ajax_failure("Unable to remove member from collection.", data);
	})
	.always(function(data) {
		refresh_members();
		add_alert("Success!", "Successfully removed user from collection.", "success");
	});
};

$(function() {
    // Replace link for collection
	$("#collection_link").attr("href", "/collection.html?collection_id=" + collection_id);
	
	// Populate list of collection members
	refresh_members();

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

	// Populate list of pending invitations
	refresh_invitations();
});

// Send invite clicked
$("#send_invite_button").click(function() {
	send_invite = true;
	$("#invite_modal").modal("hide");
});

$('#invite_modal').on('hidden.bs.modal', function (e) {
    if (send_invite) {
        $("#invite_wait").modal("show");
    }
});
$('#invite_wait').on('shown.bs.modal', function (e) {
	let payload = {
		invitee_email: $("#recipient_email").val(),
		invitee_name: $("#recipient_name").val(),
		admin_invite: $('#admin_invite').is(":checked"),
		message: $("#message").val(),
	};
	console.log("Sending invite...");
	console.log(payload)
    $.post(`/collections/${collection_id}/invitations`, JSON.stringify(payload))
    .done(function(data) {
        console.log("Invitation sent.");
		console.log(data);
		add_alert("Invitation sent!", `The invitation has been sent to ${payload.invitee_email}.`, "success");
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to send invite.", data);
    })
    .always(function() {
		refresh_invitations();
        $("#invite_wait").modal("hide");
        send_invite = false;
    });
});

// Logout button
$("#logout").click(function() {
	$.get("/user/logout");
	window.location.href = "/"
});