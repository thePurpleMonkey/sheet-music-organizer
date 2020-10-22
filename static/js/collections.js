"use strict";

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
			window.location.replace("/signin.html");
		}
		alert("Unable to get collections.\n" + data.responseJSON.error);
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
		$(".alert").show();
	})
	.fail(function(data) {
		alert("Unable to create collection.\n" + data.responseJSON.error);
	})
	.always(function() {
		create_collection = false;
		$("#wait").modal("hide");
		refreshCollections();
	});
});

// Close alert
$("#alert-close").click(function() {
	$(".alert").hide();
});

// Logout button
$("#logout").click(function() {
	$.get("/user/logout");
	window.location.href = "/"
});