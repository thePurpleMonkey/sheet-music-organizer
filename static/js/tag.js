"use strict";

import { add_alert, alert_ajax_failure } from "./utilities.js";

let url = new URL(window.location.href);
let delete_tag = undefined;
let songs = [];
let edit_mode = false;
let save_tag = false;

let tag = {
    // Parse name and collection ID from URL parameter
    tag_id: url.searchParams.get("tag_id"),
	collection_id: url.searchParams.get("collection_id"),
	

    // These attributes get set after an AJAX call to server
	name: undefined,
    description: undefined,
};

// Show options in navbar
$("#navbar_options").show();
$("#navbar_dashboard").show();

function refresh_tag() {
    $.get(`/collections/${tag.collection_id}/tags/${tag.tag_id}`)
    .done(function(data) {
		console.log("Loading tag result:");
        console.log(data);
        tag.tag_id = data.tag_id;
        tag.name = data.name;
        tag.description = data.description;
        
        console.log("Tag:");
        console.log(tag);

        $("#page_header").text(tag.name);
        if (tag.description) {
            $("#description").text(tag.description);
        } else {
            $("#description_row").hide();
        }
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to get tag information.", data);
    });
}

// Get tag info when document becomes ready
$(function() {
    // Replace link for collection
    $("#collection_link").attr("href", "/collection.html?collection_id=" + tag.collection_id);

    // Load tag information
    refresh_tag();
    
    // Get songs tagged with this tag
    $.get(`/collections/${tag.collection_id}/tags/${tag.tag_id}/songs`)
    .done(function(data) {
		console.log("Loading tagged songs result:");
        console.log(data);

        songs = data;
        
        console.log("Songs:");
        console.log(songs);

        songs.forEach(song => {
            let element = $("<a>")
            .attr("href", `song.html?collection_id=${tag.collection_id}&song_id=${song.song_id}`)
            .addClass("list-group-item list-group-item-action")
            .text(song.name);

            $("#songs_container").append(element);
        });
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to get songs.", data);
    });
});

// Cancel edits
$("#edit_cancel").click(function() {
    $("#page_header").text(tag.name);
    $("#tag_name").val(tag.name);
    $("#tag_description").val(tag.description);
    
    // Disable inputs
	$("#tag_name").prop("disabled", true);
	$("#tag_description").prop("disabled", true);
    $("#edit_buttons").hide(500);
})

// Saves changes to tag
$("#edit_button").click(function() {
    edit_mode = true;

    $("#tag_name").val(tag.name);
    $("#tag_description").val(tag.description);
    $("#edit_tag_modal").modal("show");
});
$("#save_tag_button").click(function() {
    save_tag = true;
    $("#edit_tag_modal").modal("hide");
});
$("#edit_tag_modal").on("hidden.bs.modal", function (e) {
    if (save_tag) {
        $("#edit_tag_wait").modal("show");
    }
});
$('#edit_tag_wait').on('shown.bs.modal', function (e) {
    let payload = JSON.stringify({
		name: $("#tag_name").val(), 
		description: $("#tag_description").val(),
    });
    console.log("Updating tag: " + payload);
    $.ajax({
        method: "PUT",
        url: `/collections/${tag.collection_id}/tags/${tag.tag_id}`,
        data: payload,
        headers: {
            "Content-Type": "application/json"
        }
    })
    .done(function(data) {
		console.log("Edit tag result:");
        console.log(data);
        add_alert("Changes saved.", "Changes to this tag have been saved.", "success");
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to save tag.", data);
    })
    .always(function() {
        $("#edit_tag_wait").modal("hide");
        refresh_tag();
    });
});

// #region Delete tag
$("#delete_button").click(function() {
    $("#delete_tag_modal").modal("show");
});
$("#delete_tag").click(function() {
    delete_tag = true;
    $("#delete_tag_modal").modal("hide");
});
$('#delete_tag_modal').on('hidden.bs.modal', function (e) {
    if (delete_tag) {
        $("#delete_tag_wait").modal("show");
    }
});
$('#delete_tag_wait').on('shown.bs.modal', function (e) {
    $.ajax({
        method: "DELETE",
        url: `/collections/${tag.collection_id}/tags/${tag.tag_id}`
    })
    .done(function(data) {
        console.log("Tag delete.");
        console.log(data);
        window.location.replace("/collection.html?collection_id=" + tag.collection_id);
    })
    .fail(function(data) {
        $("#delete_tag_wait").modal("hide");
        alert_ajax_failure("Unable to delete tag!", data);
    })
    .always(function() {
        delete_tag = false;
    });
});
// #endregion
