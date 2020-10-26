"use strict";

import { add_alert } from "./utilities.js";

let url = new URL(window.location.href);
let delete_song = false;
let add_tag = false;
let delete_tag = undefined;
let edit_mode = false;

let song = {
    // Parse name and collection ID from URL parameter
    song_id: url.searchParams.get("song_id"),
	collection_id: url.searchParams.get("collection_id"),
	

    // These attributes get set after an AJAX call to server
    name: undefined,
    artist: undefined,
    date_added: undefined,
    location: undefined,
    last_performed: undefined,
    notes: undefined,
    added_by: undefined
};

function tag_button_handler(e)
{
    let tag_id = $(e.target).data("tag_id");

    if (edit_mode) {
        $("#delete_tag_wait_modal").modal("show");
        delete_tag = tag_id;
    } else {
        // Link to tag
        window.location.href = `tag.html?tag_id=${tag_id}&collection_id=${song.collection_id}`;
    }
}

// Get all tags for this song
function refresh_tags() {
    $.get(`/collections/${song.collection_id}/songs/${song.song_id}/tags`)
    .done(function(song_tags) {
        console.log("Tags this song has been tagged with:");
        console.log(song_tags);

        // Add tags to page
        $("#tag_container").children().not("#add_tag_button").remove();
        if (song_tags != null) {
            song_tags.forEach(tag => {
                let tag_button = $("<button>")
                    .attr("type", "button")
                    .data("tag_id", tag.tag_id)
                    .addClass("btn btn-secondary")
                    .text(tag.name)
                    .click(tag_button_handler);
                $("#tag_container").children().last().before(tag_button);
            });
        }

        // Calculate what tags can be added to this song
        refresh_available_tags(song_tags);
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to get song tags information!", data);
    });
}

// Populate the list of tags that can be added to a song
function refresh_available_tags(song_tags) {
    $.get(`/collections/${song.collection_id}/tags`)
    .done(function(data) {
        let used_tag_ids = [];
        let available_tags = [];

        console.log("All tags in this collection:");
        console.log(data);

        if (song_tags == null) {
            song_tags = [];
        }

        // Get a list of tag IDs that have already been used
        song_tags.forEach(tag => used_tag_ids.push(tag.tag_id));

        // Generate a list of tags that are have not already been used
        data.forEach(tag => {
            if (!used_tag_ids.includes(tag.tag_id)) {
                available_tags.push(tag);
            }
        });

        console.log("Available tags:");
        console.log(available_tags);
        
        // Add tags to option list
        $("#tag_list").empty();
        $("#tag_list").append($("<option>").attr("value", "").attr("selected", "selected").attr("disabled", true).attr("hidden", true).text("Please select a tag"));
        available_tags.forEach(item => {
            $("#tag_list").append($("<option>").attr("value", item.tag_id).text(item.name))
        });
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to get your tags!", data);
    });
}

// Get song info when document becomes ready
$(function() {
    // Replace link for collection
    $("#collection_link").attr("href", "/collection.html?collection_id=" + song.collection_id);

    // Handler for .ready() called.
    $.get(`/collections/${song.collection_id}/songs/${encodeURIComponent(song.song_id)}`)
    .done(function(data) {
		console.log("Loading song result:");
        console.log(data);
        song.name = data.name;
        song.artist = data.artist;
        song.date_added = new Date(data.date_added);
        song.location = data.location;
        // song.last_performed = new Date(data.last_performed);
        song.notes = data.notes;
        song.added_by = data.added_by;

        if ("last_performed" in data) {
            song.last_performed = new Date(data.last_performed);
        }
        console.log("Song:");
        console.log(song);

		$("#page_header").text(song.name);
		$("#song_name").val(song.name);
		$("#song_artist").val(song.artist);
		$("#song_date_added").val(song.date_added.toISOString().substring(0, 10));
		$("#song_location").val(song.location);
		$("#song_notes").val(song.notes);
        $("#song_added_by").val(song.added_by);

        if (song.last_performed) {
            $("#song_last_performed").val(song.last_performed.toISOString().substring(0, 10));
        }
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to get song information!", data);
    });

    // Load tags for song
    refresh_tags();
});

// Cancel edits
$("#edit_cancel").click(function() {
    edit_mode = false;
    $("#page_header").text(song.name);
    $("#song_name").val(song.name);
    $("#song_artist").val(song.artist);
    $("#song_date_added").val(song.date_added.toISOString().substring(0, 10));
    $("#song_location").val(song.location);
    if (song.last_performed)
        $("#song_last_performed").val(song.last_performed.toISOString().substring(0, 10));
    $("#song_notes").val(song.notes);
    $("#song_added_by").val(song.added_by);
    
    // Disable inputs
    set_editing_mode(false);
});

// Add tag button clicked
$("#add_tag").click(function() {
    add_tag = true
    $("#add_tag_modal").modal("hide");
});

// Show tag wait dialog after add tag modal is closed
$('#add_tag_modal').on('hidden.bs.modal', function (e) {
    if (add_tag) {
        $("#tag_wait").modal("show");
    }
});
$('#tag_wait').on('shown.bs.modal', function (e) {
    let payload = JSON.stringify({song_id: parseInt(song.song_id, 10), tag_id: parseInt($("#tag_list").val(), 10)});
    $.post(`/collections/${song.collection_id}/songs/${song.song_id}/tags`, payload)
    .done(function(data) {
        console.log(data);
        add_alert("Tag created!", "The tag was successfully created. You may now start tagging your songs with it.", "success");
        refresh_tags();
        set_editing_mode(false);
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to add tag.", data);
    })
    .always(function() {
        add_tag = false;
        $("#tag_wait").modal("hide");
    });
});

function set_editing_mode(is_editing) {
    edit_mode = is_editing;
	$("#song_artist").prop("disabled", !is_editing);
	$("#song_location").prop("disabled", !is_editing);
	$("#song_last_performed").prop("disabled", !is_editing);
    $("#song_notes").prop("disabled", !is_editing);

    if (is_editing) {
        $("#edit_buttons").show(500);
        $("#add_tag_button").show(500);
    
        $("#tag_container").children().not($("#add_tag_button")).addClass("deletable");
    } else {
        $("#edit_buttons").hide(500);
        $("#add_tag_button").hide(500);

        $("#tag_container").children().not($("#add_tag_button")).removeClass("deletable");
    }
}


// Saves changes to song
$("#edit_button").click(function() {
    set_editing_mode(true);
});
$("#song_save").click(function() {
    $("#edit_song_wait").modal("show");
});
$('#edit_song_wait').on('shown.bs.modal', function (e) {
    let payload = JSON.stringify({
        name: $("#song_name").val(),
		artist: $("#song_artist").val(), 
		//date_added: $("#song_date_added").val(),
		location: $("#song_location").val(),
		last_performed: $("#song_last_performed").val(),
		notes: $("#song_notes").val(),
		added_by: $("#song_added_by").val()
	});
    $.ajax({
        method: "PUT",
        url: `/collections/${song.collection_id}/songs/${song.song_id}`,
        data: payload,
        headers: {
            "Content-Type": "application/json"
        }
    })
    .done(function(data) {
		console.log("Edit song result:");
        console.log(data);
        add_alert("Changes saved!", "Changes to this song have been successfully saved.");
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to save song.", data);
    })
    .always(function() {
        $("#edit_song_wait").modal("hide");
        set_editing_mode(false);
    });
});

// Delete song
$("#delete_song").click(function() {
    delete_song = true;
    $("#delete_song_modal").modal("hide");
});
$('#delete_song_modal').on('hidden.bs.modal', function (e) {
    if (delete_song) {
        $("#delete_song_wait").modal("show");
    }
});
$('#delete_song_wait').on('shown.bs.modal', function (e) {
    $.ajax({
        method: "DELETE",
        url: `/collections/${song.collection_id}/songs/${song.name}`
    })
    .done(function(data) {
        console.log("Song delete.");
        console.log(data);
        window.location.replace("/collection.html?collection_id=" + song.collection_id);
    })
    .fail(function(data) {
        $("#delete_song_wait").modal("hide");
        alert_ajax_failure("Unable to delete song!", data);
    })
    .always(function() {
        delete_song = false;
    });
});

// Remove tag
$('#delete_tag_wait_modal').on('shown.bs.modal', function (e) {
    let payload = JSON.stringify({
        tag_id: delete_tag
    });
    $.ajax({
        method: "DELETE",
        url: `/collections/${song.collection_id}/songs/${song.song_id}/tags`,
        data: payload,
        headers: {
            "Content-Type": "application/json"
        }
    })
    .done(function(data) {
        console.log("Remove tag result:");
        console.log(data);
        add_alert("Tag removed!", "The tag was successfully removed.", "success");

    })
    .fail(function(data) {
        alert_ajax_failure("Unable to remove tag.", data);
    })
    .always(function() {
        $("#delete_tag_wait_modal").modal("hide");
        delete_tag = undefined;
        refresh_tags();
    });
});

// Logout button
$("#logout").click(function() {
    $.get("/user/logout");
    window.location.href = "/"
});