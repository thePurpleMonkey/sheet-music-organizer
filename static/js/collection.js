"use strict";

import { add_alert, getUrlParameter, alert_ajax_failure } from "./utilities.js";

let add_song = false;
let add_tag = false;
let edit_collection = false;
let delete_collection = false;

let collection = {
    // Parse collection ID from URL parameter
    id: getUrlParameter("collection_id"),

    // These attributes get set after an AJAX call to server
    name: undefined,
    description: undefined
};

// Show options in navbar
$("#navbar_options").show();
$("#navbar_members").show();

// Get collection info when document becomes ready
$(function() {
    // Replace link for members
    $("#members_link").attr("href", "/members.html?collection_id=" + collection.id);
    
    // Handler for .ready() called.
    $.get(`/collections/${collection.id}`)
    .done(function(data) {
        console.log(data);
        collection.name = data.name;
        collection.description = data.description;

        $("#page_header").text(collection.name);
        $("#collection_name").val(collection.name);
        $("#collection_description").val(collection.description);

        if (collection.description) {
            $("#description").text(collection.description);
        } else {
            $("#description_header").hide();
        }
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to get collection information!", data);
    });

    reloadSongs();
    reloadTags();
});

function reloadSongs() {
    $("#songs").empty();
    $("#songs").append('<a class="list-group-item">Loading songs, please wait...</a>');
    $.get(`/collections/${collection.id}/songs`)
    .done(function(data) {
        console.log("Get songs result:");
        console.log(data);
        $("#songs").empty();

        data.forEach(song => {
            let a = $("<a>");
            a.addClass("list-group-item");
            a.addClass("list-group-item-action");
            a.attr("href", `/song.html?song_id=${encodeURIComponent(song.song_id)}&collection_id=${collection.id}`);
            a.text(song.name);
            $("#songs").append(a);
        });
    })
    .fail(function(data) {
        if (data.status == 403) {
            window.location.replace("/404.html");
        }

        alert_ajax_failure("Unable to get songs!", data);
    })
    .always(function() {
        $("#loading").remove();
    });
};

function reloadTags() {
    $("#tags").empty();
    $("#tags").append('<a class="list-group-item">Loading tags, please wait...</a>');
    $.get(`/collections/${collection.id}/tags`)
    .done(function(data) {
        console.log("Get tags result:");
        console.log(data);
        $("#tags").empty();

        data.forEach(tag => {
            let a = $("<a>");
            a.addClass("list-group-item");
            a.addClass("list-group-item-action");
            a.attr("href", `/tag.html?tag_id=${encodeURIComponent(tag.tag_id)}&collection_id=${collection.id}`);
            a.text(tag.name);
            $("#tags").append(a);
        });
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to get tags!", data);
    })
    .always(function() {
        $("#loading").remove();
    });
};

// Make Song POST API call after wait dialog is shown
$('#song_wait').on('shown.bs.modal', function (e) {
    let payload = JSON.stringify({
        name: $("#name").val(), 
        artist: $("#artist").val(),
        location: $("#location").val(),
        notes: $("#notes").val(),
    });

    let last_performed = $("#last_performed").val();
    if (last_performed !== "") {
        payload.last_performed = new Date(last_performed).toISOString();
    }

    $.post(`/collections/${collection.id}/songs`, payload)
    .done(function(data) {
        console.log(data);
        add_alert("Song added!", "The song was successfully added to your collection.", "success");
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to add song!", data);
    })
    .always(function() {
        add_song = false;
        $("#song_wait").modal("hide");
        reloadSongs();
    });
});

// Make tag POST API call after tag wait dialog is shown
$('#tag_wait').on('shown.bs.modal', function (e) {
    let payload = JSON.stringify({
        name: $("#tag_name").val(),
        description: $("#tag_description").val(),
    });
    console.log("Adding tag: " + payload);
    $.post(`/collections/${collection.id}/tags`, payload)
    .done(function(data) {
        console.log(data);
        add_alert("Tag created!", "The tag was successfully created. You may now start tagging your songs with it.", "success");
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to add tag!", data);
    })
    .always(function() {
        $("#tag_wait").modal("hide");
        reloadTags();
    });
});

// Saves changes to collection
$("#save_collection").click(function() {
    edit_collection = true;
    $("#edit_collection_modal").modal("hide");
});
$('#edit_collection_modal').on('hidden.bs.modal', function (e) {
    if (edit_collection) {
        $("#edit_collection_wait").modal("show");
    }
});
$('#edit_collection_wait').on('shown.bs.modal', function (e) {
    let payload = JSON.stringify({name: $("#collection_name").val(), description: $("#collection_description").val()});
    $.ajax({
        url: `/collections/${collection.id}`,
        type: 'PUT',
        data: payload,
        // success: function(result) {
        //     // Do something with the result
        // }
    })
    .done(function(data) {
        console.log(data);
        add_alert("Changes saved!", "The changes you made to the collection were successfully saved.", "success");
        $("#page_header").text($("#collection_name").val());
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to save collection.", data);
    })
    .always(function() {
        edit_collection = false;
        $("#edit_collection_wait").modal("hide");
    });
});

// Delete collection
$("#delete_collection").click(function() {
    delete_collection = true;
    $("#delete_collection_modal").modal("hide");
});
$('#delete_collection_modal').on('hidden.bs.modal', function (e) {
    if (delete_collection) {
        $("#delete_collection_wait").modal("show");
    }
});
$('#delete_collection_wait').on('shown.bs.modal', function (e) {
    $.ajax({
        method: "DELETE",
        url: `/collections/${collection.id}`
    })
    .done(function(data) {
        console.log("Collection delete.");
        console.log(data);
        window.location.replace("/collections.html");
    })
    .fail(function(data) {
        delete_collection = false;
        $("#delete_collection_wait").modal("hide");
        alert_ajax_failure("Unable to delete collection!", data);
    });
});

// Show wait dialog after add song modal is closed
$("#add_song_modal_button").click(function() {
    add_song = true;
    $("#add_song_modal").modal("hide");
});
$('#add_song_modal').on('hidden.bs.modal', function (e) {
    if (add_song) {
        $("#song_wait").modal("show");
    }
});

// Show tag wait dialog after add tag modal is closed
$('#add_tag_modal').on('hidden.bs.modal', function (e) {
    if (add_tag) {
        $("#tag_wait").modal("show");
    }
});
$("#add_tag_modal_button").click(function() {
    add_tag = true
    $("#add_tag_modal").modal("hide");
});
