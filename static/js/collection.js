"use strict";

let add_song = false;
let add_tag = false;
let edit_collection = false;
let delete_collection = false;

// Code snippet pulled from https://stackoverflow.com/questions/19491336/get-url-parameter-jquery-or-how-to-get-query-string-values-in-js
var getUrlParameter = function getUrlParameter(sParam) {
    var sPageURL = window.location.search.substring(1),
            sURLVariables = sPageURL.split('&'),
            sParameterName,
            i;

    for (i = 0; i < sURLVariables.length; i++) {
        sParameterName = sURLVariables[i].split('=');

        if (sParameterName[0] === sParam) {
                return sParameterName[1] === undefined ? true : decodeURIComponent(sParameterName[1]);
        }
    }
};

let collection = {
    // Parse collection ID from URL parameter
    id: getUrlParameter("collection_id"),

    // These attributes get set after an AJAX call to server
    name: undefined,
    description: undefined
};

// Get collection info when document becomes ready
$(function() {
    // Handler for .ready() called.
    $.get(`/collections/${collection.id}`)
    .done(function(data) {
        console.log(data);
        collection.name = data.name;
        collection.description = data.description;

        $("#collection_name").val(collection.name);
        $("#collection_description").val(collection.description);
    })
    .fail(function(data) {
        alert("Unable to get collection information.\n" + data.responseJSON.error);
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
            a.attr("href", `/song.html?name=${encodeURIComponent(song.name)}&collection_id=${collection.id}`);
            a.text(song.name);
            $("#songs").append(a);
        });
    })
    .fail(function(data) {
        if (data.status == 403) {
            window.location.replace("/404.html");
        }
        alert("Unable to get songs.\n" + data.responseJSON.error);
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

        data.forEach(song => {
            let a = $("<a>");
            a.addClass("list-group-item");
            a.addClass("list-group-item-action");
            a.attr("href", `/tag.html?name=${encodeURIComponent(song.name)}&collection_id=${collection.id}`);
            a.text(song.name);
            $("#tags").append(a);
        });
    })
    .fail(function(data) {
        alert("Unable to get tags.\n" + data.responseJSON.error);
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
        last_performed: $("#last_performed").val(),
        notes: $("#notes").val(),
    });
    $.post(`/collections/${collection.id}/songs`, payload)
    .done(function(data) {
        console.log(data);
        $("#song_added_alert").show();
    })
    .fail(function(data) {
        alert("Unable to add song.\n" + data.responseJSON.error);
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
        notes: $("#tag_description").val(),
    });
    $.post(`/collections/${collection.id}/tags`, payload)
    .done(function(data) {
        console.log(data);
        $("#tag_added_alert").show();
    })
    .fail(function(data) {
        alert("Unable to add tag.\n" + data.responseJSON.error);
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
    $.post(`/collections/${collection.id}`, payload)
    .done(function(data) {
        console.log(data);
        $("#edit_collection_alert").show();
    })
    .fail(function(data) {
        alert("Unable to save collection.\n" + data.responseJSON.error);
    })
    .always(function() {
        edit_collection = false;
        $("#edit_collection_wait").modal("hide");
    });
});

// Delete collection collection
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
        alert("Unable to delete collection!\n" + data.responseJSON.error);
    });
});

// Show wait dialog after add song modal is closed
$("#add_song").click(function() {
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
    $("#tag_wait").modal("show");
});
$("#add_tag").click(function() {
    $("#add_tag_modal").modal("hide");
});

// Close alerts
$("#alert-close").click(function() {
    $("#song_added_alert").hide()
});
$("#edit_alert_close").click(function() {
    $("#edit_collection_alert").hide()
});

// Logout button
$("#logout").click(function() {
    $.get("/user/logout");
    window.location.href = "/"
});