"use strict";

import { alert_ajax_failure } from "./utilities.js";

let url = new URL(window.location.href);
let songs = [];

let setlist = {
    // Parse share code from URL parameter
    share_code: url.searchParams.get("code"),

    // These attributes get set after an AJAX call to server
    setlist_id: undefined,
	collection_id: undefined,
    name: undefined,
    date: undefined,
    notes: undefined,
    shared: undefined,
};

// Hide navbar links
$("#navbar_collections").addClass("hidden");

function refresh_setlist() {
    $.get(`/setlists/${setlist.share_code}`)
    .done(function(data) {
		console.log("Loading setlist result:");
        console.log(data);
        setlist.setlist_id = data.setlist_id;
        setlist.name = data.name;
        setlist.date = data.date ? new Date(data.date) : undefined;
        setlist.notes = data.notes;
        setlist.shared = data.shared;
        setlist.share_code = data.share_code;
        
        console.log("Setlist:");
        console.log(setlist);

        // Update page UI
        $("#page_header").text(setlist.name);
        document.title = `${setlist.name} - Setlist - Sheet Music Organizer`;
        setlist.date ? $("#setlist_date").text(setlist.date.toISOString().substring(0, 10)) : $("#setlist_date").html("&nbsp;");
        setlist.notes ? $("#setlist_notes").text(setlist.notes) : $("#setlist_notes").html("&nbsp;");
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to get setlist information.", data);
    });
}

function refresh_setlist_songs() {
    // Get songs in this setlist
    $.get(`/setlists/${setlist.share_code}/songs`)
    .done(function(data) {
		console.log("Loading songs result:");
        console.log(data);

        songs = data;
        
        console.log("Songs:");
        console.log(songs);

        // Clear existing songs
        $("#songs_container").empty();

        if (songs.length === 0) {
            $("#songs_container").html("&nbsp;");
        } else {
            songs.sort((a, b) => a.order - b.order).forEach(song => {
    
                let element = $("<div>")
                .addClass("list-group-item")
                .text(song.name);
    
                $("#songs_container").append(element);
            });
        }
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to get songs.", data);
    });
}

// Startup function
$(function() {
    // Load setlist information
    refresh_setlist();
    refresh_setlist_songs();
});
