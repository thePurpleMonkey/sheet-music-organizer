"use strict";

import { add_alert, alert_ajax_failure, getUrlParameter } from "./utilities.js";

let collection_id = getUrlParameter("collection_id");
let send_invite = false;
let leave = false;
let current_user_id;
let members = [];

let datetime_format = new Intl.DateTimeFormat([], {
	dateStyle: "short",
	timeStyle: "short"
});

// Hide options in navbar
$("#navbar_dashboard").removeClass("hidden");
$("#navbar_options").removeClass("hidden");
$("#navbar_member_options").removeClass("hidden");
$("#navbar_edit_options").addClass("hidden");

// Enable tooltips


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
	$("#administrators_list").empty();
	$("#users_list").empty();

	$("#members_list").append("<li>Loading members, please wait...</li>");
	$.get(`/collections/${collection_id}/members`)
	.done(function(data) {
		console.log("Members list:");
		console.log(data);
		$("#members_list").empty();

		current_user_id = data.user_id;
		members = data.members;
		let admin = data.admin;

		if (admin) {
			$("#invite_button").removeClass("hidden");
			$("#manage_members_button").removeClass("hidden");
		} else {
			$("#invite_button").addClass("hidden");
			$("#manage_members_button").addClass("hidden");
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

			if (current_user_id === member.user_id) {
				member.self = true;
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
				$("#administrators_list").append(item);
			}

			$("#members_list").append(item);

			// Add user to appropriate manager user modal list
			item = $("<button type='button'>").addClass("list-group-item list-group-item-action").text(member.name);
			if (member.self) {
				item.attr("disabled", true).attr("style", "pointer-events: none;")
				let span = $("<span>")
				.attr("title", "You cannot remove your own administrator status!")
				.attr("data-toggle", "tooltip");

				console.log(span);

				span.append(item);
				span.tooltip({
					trigger: "click"
				});
				span.on("shown.bs.tooltip", function(e) {
					setTimeout(function () {
						$(e.target).tooltip('hide');
					  }, 2000);
				})
				$("#administrators_list").append(span);
				return; // Continue to the next iteration of the loop
			}
			
			// Store user_id with this element
			item.data("user_id", member.user_id);
			item.data("name", member.name);

			if (member.admin) {
				item.data("admin", false);
				item.prepend(
					$("<img>")
					.attr("src", "/img/user-x.svg")
					.attr("alt", "Remove admin icon")
					.attr("title", "This member will lose their admin privileges")
					.addClass("admin-icon hidden")
				);
				item.click(function() {
					if (!$(this).attr("disabled")) {
						$(this).toggleClass("list-group-item-warning selected");
						$(this).find("img").toggleClass("hidden");
					}
				});
				$("#administrators_list").append(item);
			} else {
				item.data("admin", true);
				item.prepend(
					$("<img>")
					.attr("src", "/img/user-plus.svg")
					.attr("alt", "Add admin icon")
					.attr("title", "This member will be promoted to admin")
					.addClass("admin-icon hidden")
				);
				item.click(function() {
					if (!$(this).attr("disabled")) {
						$(this).toggleClass("list-group-item-success selected");
						$(this).find("img").toggleClass("hidden");
					}
				});
				$("#users_list").append(item);
			}
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

// #region Invite
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
		console.log("Invite POST response:");
		console.log(data);
		if (data.responseJSON.code === "unverified") {
			// User is unverified
			add_alert("Unverified account", "You need to verify your account before you can send invitations. Please visit your <a href='/account.html' class='alert-link'>account page</a> to verify your account.", "danger");
		} else {
			// Something else went wrong
        alert_ajax_failure("Unable to send invite.", data);
		}
    })
    .always(function() {
		refresh_invitations();
        $("#invite_wait").modal("hide");
        send_invite = false;
    });
});
// #endregion

// #region Leave
$("#leave_modal_button").click(function() {
	leave = true;
	$("#leave_modal").modal("hide");
});
$('#leave_modal').on('hidden.bs.modal', function (e) {
    if (leave) {
        $("#leave_wait").modal("show");
    }
});
$('#leave_wait').on('shown.bs.modal', function (e) {
	// Make AJAX request
	$.ajax(`/collections/${collection_id}/members/${current_user_id}`, {
		method: "DELETE"
	})
	.done(function(data) {
		console.log("Leave collection response:")
		console.log(data);
		if (add_session_alert("Success!", "Successfully left collection.", "success")) {
			window.location.href = "/collections.html";
		}
	})
	.fail(function(data) {
		alert_ajax_failure("Unable to leave collection.", data);
	})
	.always(function() {
		$("#leave_wait").modal("hide");
	});
});
// #endregion

// #region Manage
$("#save_admins_button").click(function() {
	if ($(".selected").length > 0) {
		leave = true;
	}
	$("#manage_modal").modal("hide");
});
$('#manage_modal').on('hidden.bs.modal', function (e) {
    if (leave) {
        $("#manage_wait").modal("show");
    }
});
$('#manage_wait').on('shown.bs.modal', function (e) {
	// Get all modified users
	let members = $(".selected");

	let requests = [];

	members.each(function() {
		let member = $(this);
		let name = member.data("name");

		let payload = {
			admin: member.data("admin")
		}
		console.log(`Member ${member.data("user_id")}:`)
		console.log(payload)
		// Make AJAX request
		requests.push(
			$.ajax(`/collections/${collection_id}/members/${member.data("user_id")}`, {
				method: "PUT",
				data: JSON.stringify(payload),
			})
			.done(function() {
				add_alert("Saved changes", `Successfully saved changes for "${name}".`, "success");
			})
			.fail(function(data) {
				alert_ajax_failure(`Unable to save changes for "${name}".`, data);
			})
		);
	});

	$.when(requests).always(function() {
		refresh_members();
		$("#manage_wait").modal("hide");
		leave = false;
	})
});
// #endregion
