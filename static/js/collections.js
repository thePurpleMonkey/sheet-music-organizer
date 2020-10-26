"use strict";

import { add_alert, alert_ajax_failure } from "./utilities.js";

let create_collection = false;

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

refreshCollections();

$("#create_collection").click(function() {
	create_collection = true;
	$("#new_collection_modal").modal("hide");
});

$('#new_collection_modal').on('hidden.bs.modal', function (e) {
	$("#wait").modal("show");
});

$('#wait').on('shown.bs.modal', function (e) {
	let payload = JSON.stringify({name: $("#name").val(), description: $("#description").val()});
	$.post("/collections", payload)
	.done(function(data) {
		console.log(data);
		add_alert("Collection created!", "The collection was successfully created.", "success");
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

// Logout button
$("#logout").click(function() {
	$.get("/user/logout");
	window.location.href = "/"
});