"use strict";

let url = new URL(window.location.href);
let delete_song = false;

let song = {
    // Parse name and collection ID from URL parameter
	name: url.searchParams.get("name"),
	collection_id: url.searchParams.get("collection_id"),
	

    // These attributes get set after an AJAX call to server
    artist: undefined,
    date_added: undefined,
    location: undefined,
    last_performed: undefined,
    notes: undefined,
    added_by: undefined
};

// Get song info when document becomes ready
$(function() {
    // Replace link for collection
    $("#collection_link").attr("href", "/collection.html?collection_id=" + song.collection_id);

    // Handler for .ready() called.
    $.get(`/collections/${song.collection_id}/songs/${encodeURIComponent(song.name)}`)
    .done(function(data) {
		console.log("Loading song result:");
        console.log(data);
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
        alert("Unable to get song information.\n" + data.responseJSON.error);
    });
});

// Cancel edits
$("#edit_cancel").click(function() {
    $("#page_header").text(song.name);
    $("#song_name").val(song.name);
    $("#song_artist").val(song.artist);
    $("#song_date_added").val(song.date_added.toISOString().substring(0, 10));
    $("#song_location").val(song.location);
    $("#song_last_performed").val(song.last_performed.toISOString().substring(0, 10));
    $("#song_notes").val(song.notes);
    $("#song_added_by").val(song.added_by);
    
    // Disable inputs
	$("#song_artist").prop("disabled", true);
	$("#song_location").prop("disabled", true);
	$("#song_last_performed").prop("disabled", true);
    $("#song_notes").prop("disabled", true);
    $("#edit_buttons").hide(500);
})

// Saves changes to song
$("#edit_button").click(function() {
	$("#song_artist").prop("disabled", false);
	$("#song_location").prop("disabled", false);
	$("#song_last_performed").prop("disabled", false);
    $("#song_notes").prop("disabled", false);
    $("#edit_buttons").show(500);
});
$("#song_save").click(function() {
    $("#edit_song_wait").modal("show");
});
$('#edit_song_wait').on('shown.bs.modal', function (e) {
    let payload = JSON.stringify({
		artist: $("#song_artist").val(), 
		date_added: $("#song_date_added").val(),
		location: $("#song_location").val(),
		last_performed: $("#song_last_performed").val(),
		notes: $("#song_notes").val(),
		added_by: $("#song_added_by").val()
	});
    $.ajax({
        method: "PUT",
        url: `/collections/${song.collection_id}/songs/${encodeURIComponent(song.name)}`,
        data: payload,
        headers: {
            "Content-Type": "application/json"
        }
    })
    .done(function(data) {
		console.log("Edit song result:");
        console.log(data);
        $("#edit_song_alert").show();

    })
    .fail(function(data) {
        alert("Unable to save song.\n" + data.responseJSON.error);
    })
    .always(function() {
        $("#edit_song_wait").modal("hide");
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
        alert("Unable to delete song!\n" + data.responseJSON.error);
    })
    .always(function() {
        delete_song = false;
    });
});

// Close alerts
$("#alert-close").click(function() {
    $("#song_added_alert").hide()
});
$("#edit_alert_close").click(function() {
    $("#edit_song_alert").hide()
});

// Logout button
$("#logout").click(function() {
    $.get("/user/logout");
    window.location.href = "/"
});