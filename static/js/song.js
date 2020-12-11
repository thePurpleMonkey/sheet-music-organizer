"use strict";

import { add_alert, alert_ajax_failure, get_youtube_video_id, substitute_URLs } from "./utilities.js";

let url = new URL(window.location.href);
let delete_song = false;
let add_tag = false;
let delete_tag = undefined;
let edit_mode = false;

// Show options in navbar
$("#navbar_options").removeClass("hidden");
$("#navbar_dashboard").removeClass("hidden");
$("#search_form").removeClass("hidden");

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
                if (edit_mode) {
                    tag_button.addClass("deletable");
                }
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
        available_tags.forEach(item => {
            let button = $("<button type='button' class='btn btn-light'>")
                            .text(item.name)
                            .data("tag_id", item.tag_id)
                            .click(tag_button_clicked);
            $("#tag_list").append(button);
        });
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to get your tags!", data);
    });
}

function tag_button_clicked(e) {
    let tag_id = $(this).data("tag_id");
    
    $(this).toggleClass("btn-light");
    $(this).toggleClass("btn-success");
}

function refresh_song() {
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

        // Set page labels
		$("#page_header").text(song.name);
		$("#song_name").text(song.name);
		song.artist ? $("#song_artist").text(song.artist) : $("#song_artist").html("&nbsp;");
		$("#song_date_added").text(song.date_added.toISOString().substring(0, 10));
		song.location ? $("#song_location").text(song.location) : $("#song_location").html("&nbsp;");
        song.notes ? $("#song_notes").text(song.notes) : $("#song_notes").html("&nbsp;");
        $("#song_added_by").text(song.added_by);
        song.last_performed ? $("#song_last_performed").text(song.last_performed.toISOString().substring(0, 10)) : $("#song_last_performed").html("&nbsp;");

        // Detect URL in song location
        let result = substitute_URLs(song.location);
        console.log("Result from location link substitution:");
        console.log(result);
        if (result.html) {
            $("#song_location").html(result.html);
        }

        // Detect URLs in song notes
        result = substitute_URLs(song.notes);
        console.log("Result from link substitution:");
        console.log(result);
        if (result.html) {
            $("#song_notes").html(result.html);
        }

        // Check for links to YouTube videos and show thumbnails
        $("#thumbnail_container").empty();
        result.URLs.forEach(url => {
            let video_id = get_youtube_video_id(url);
            if (video_id) {
                let img = $("<img>").attr("src", `https://img.youtube.com/vi/${video_id}/default.jpg`);
                let a = $(`<a href="${url}" target="_blank" rel="noreferrer noopener">`);
                a.append(img);
                $("#thumbnail_container").append(a);
            }
        });
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to get song information!", data);
    });
}

// Get song info when document becomes ready
$(function() {
    // Replace link for collection
    $("#collection_link").attr("href", "/collection.html?collection_id=" + song.collection_id);

    refresh_song();
    refresh_tags();
});

// Cancel edits
$("#edit_cancel").click(function() {    
    // Disable inputs
    set_editing_mode(false);
});

// #region Add tags
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
$('#tag_wait').on('shown.bs.modal', function () {
    let requests = [];

     $("#tag_list > .btn-success").each(function() {
        let tag_id = $(this).data("tag_id");
        console.log("Tag ID:");
        console.log(tag_id);
        let payload = JSON.stringify({song_id: parseInt(song.song_id, 10), tag_id: parseInt(tag_id)});

        requests.push(
            $.post(`/collections/${song.collection_id}/songs/${song.song_id}/tags`, payload)
            .done(function(data) {
                console.log("Response for adding tag " + tag_id + " to song " + song.song_id);
                console.log(data);
            })
            .fail(function(data) {
                alert_ajax_failure("Unable to add tag.", data);
            })
        );
     });

     $.when(requests)
     .done(function() {
         console.log("When done function");
         add_alert("Tags added!", "The tags were successfully added to this song.", "success");
     })
     .always(function() {
         set_editing_mode(false);
         refresh_song();
         refresh_tags();
         refresh_available_tags();
         $("#tag_wait").modal("hide");
         add_tag = false;
     });
});
// #endregion

function set_editing_mode(is_editing) {
    edit_mode = is_editing;

    if (is_editing) {
        // Update input values
        $("#song_name_input").val(song.name);
        $("#song_artist_input").val(song.artist);
        $("#song_location_input").val(song.location);
        $("#song_notes_input").val(song.notes);

        // Get correct date for the last performed input widget
        if (song.last_performed) {
            console.log("Last performed UTC Date: " + song.last_performed.getUTCDate());
            let year = song.last_performed.getUTCFullYear();
            let month = String(song.last_performed.getUTCMonth() + 1).padStart(2, '0');
            let day = String(song.last_performed.getUTCDate()).padStart(2, '0');
            let date_string = year + "-" + month + "-" + day;
            console.log("Last performed: " + date_string);
            $("#song_last_performed_input").val(date_string);
        }

        // Show edit buttons
        $("#edit_buttons").removeClass("hidden");
        $("#add_tag_button").removeClass("hidden");
        $("#edit_divider").removeClass("hidden");

        // Show form inputs
        $("#song_name_input").removeClass("hidden");
        $("#song_artist_input").removeClass("hidden");
        $("#song_last_performed_input").removeClass("hidden");
        $("#song_location_input").removeClass("hidden");
        $("#song_notes_input").removeClass("hidden");

        // Hide labels
        $("#song_name").addClass("hidden");
        $("#song_artist").addClass("hidden");
        $("#song_last_performed").addClass("hidden");
        $("#song_location").addClass("hidden");
        $("#song_notes").addClass("hidden");
    
        $("#tag_container").children().not($("#add_tag_button")).addClass("deletable");
    } else {
        // Hide input buttons
        $("#edit_buttons").addClass("hidden");
        $("#add_tag_button").addClass("hidden");
        $("#edit_divider").addClass("hidden");

        // Hide form inputs
        $("#song_name_input").addClass("hidden");
        $("#song_artist_input").addClass("hidden");
        $("#song_last_performed_input").addClass("hidden");
        $("#song_location_input").addClass("hidden");
        $("#song_notes_input").addClass("hidden");

        // Show labels
        $("#song_name").removeClass("hidden");
        $("#song_artist").removeClass("hidden");
        $("#song_last_performed").removeClass("hidden");
        $("#song_location").removeClass("hidden");
        $("#song_notes").removeClass("hidden");

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
        name: $("#song_name_input").val(),
		artist: $("#song_artist_input").val(),
		location: $("#song_location_input").val(),
		last_performed: $("#song_last_performed_input").val(),
		notes: $("#song_notes_input").val()
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
        $("#page_header").text($("#song_name").val());
        song.name = $("#song_name").val();
        song.artist = $("#song_artist").val();
        song.location = $("#song_location").val();
        song.last_performed = $("#song_last_performed").val();
        song.notes = $("#song_notes").val();
        add_alert("Changes saved!", "Changes to this song have been successfully saved.");
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to save song.", data);
    })
    .always(function() {
        $("#edit_song_wait").modal("hide");
        set_editing_mode(false);
        refresh_song();
    });
});

// #region Delete song
$("#delete_button").click(function() {
    $("#delete_song_modal").modal("show");
});
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
        url: `/collections/${song.collection_id}/songs/${song.song_id}`
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
// #endregion

// #region Remove tag
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
// #endregion