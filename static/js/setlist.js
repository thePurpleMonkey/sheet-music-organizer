"use strict";

import { add_alert, alert_ajax_failure } from "./utilities.js";

const modes = {
    NORMAL: "normal",
    REORDER: "reorder",
    REMOVE: "remove",
}

let url = new URL(window.location.href);
let delete_setlist = undefined;
let songs = [];
let all_songs = [];
let save_setlist = false;
let add_songs = false;
let share = false;
let mode = modes.NORMAL;

let setlist = {
    // Parse name and collection ID from URL parameter
    setlist_id: url.searchParams.get("setlist_id"),
	collection_id: url.searchParams.get("collection_id"),
	

    // These attributes get set after an AJAX call to server
    name: undefined,
    date: undefined,
    notes: undefined,
    shared: undefined,
    share_code: undefined,
};

// Show options in navbar
$("#navbar_options").removeClass("hidden");
$("#navbar_dashboard").removeClass("hidden");
$("#navbar_setlists").removeClass("hidden");
$("#navbar_setlist_share").removeClass("hidden");
$("#search_form").removeClass("hidden");

// Replace link for collection
$("#collection_link").attr("href", "/collection.html?collection_id=" + setlist.collection_id);
$("#setlists_link").attr("href", "/setlists.html?collection_id=" + setlist.collection_id);

function refresh_setlist() {
    $.get(`/collections/${setlist.collection_id}/setlists/${setlist.setlist_id}`)
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
        setlist.date ? $("#setlist_date").text(setlist.date.toISOString().substring(0, 10)) : $("#setlist_date").html("&nbsp;");
        setlist.notes ? $("#setlist_notes").text(setlist.notes) : $("#setlist_notes").html("&nbsp;");
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to get setlist information.", data);
    });
}

function refresh_setlist_songs() {
    // Get songs in this setlist
    $.get(`/collections/${setlist.collection_id}/setlists/${setlist.setlist_id}/songs`)
    .done(function(data) {
		console.log("Loading songs result:");
        console.log(data);

        songs = data;
        
        console.log("Songs:");
        console.log(songs);

        // Clear existing songs
        $("#songs_container").empty();
        $("#songs_container_delete").empty();
        $("#songs_container_reorder").empty();

        if (songs.length === 0) {
            // Make sure both of these divs take up the correct amount of space even though they're empty
            $("#songs_container").html("&nbsp;");
            $("#songs_container_delete").html("&nbsp;");
            $("#songs_container_reorder").html("&nbsp;");
        } else {
            songs.sort((a, b) => a.order - b.order).forEach(song => {
    
                let element = $("<a>")
                .attr("href", `song.html?collection_id=${setlist.collection_id}&song_id=${song.song_id}`)
                .addClass("list-group-item list-group-item-action")
                .attr("data-id", song.song_id)
                .text(song.name);
    
                $("#songs_container").append(element);

                // Add to delete song list
                element = $("<a>")
                    .attr("href", "javascript:;").addClass("list-group-item-action")
                    .addClass("list-group-item")
                    .text(song.name)
                    .data("song_id", song.song_id)
                    .click(remove_from_setlist)
                    .hover(function() { if(mode == modes.REMOVE) { $(this).toggleClass("list-group-item-danger"); }});
                $("#songs_container_delete").append(element);

                // Add to reorder song list
                element = $("<div>")
                    .addClass("list-group-item")
                    .text(song.name)
                    .data("song_id", song.song_id)
                    .attr("data-id", song.song_id);
                $("#songs_container_reorder").append(element);
            });
        }

        refresh_add_song_list();
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to get songs.", data);
    });
}

// Get all songs in this collection
function refresh_add_song_list() {
    $.get(`/collections/${setlist.collection_id}/songs`)
    .done(function(data) {
		console.log("Loading all collection songs result:");
        console.log(data);

        all_songs = data;
        
        console.log("All songs:");
        console.log(all_songs);

        // Clear old options before adding new ones
        $("#song_select option:not(:first-child)").remove();

        all_songs
            .sort((a, b) => a.name.localeCompare(b.name))
            .forEach(function(song) {
                let option = $("<option>")
                    .attr("value", song.song_id)
                    .text(song.name);
                $("#song_select").append(option);
                if (songs.some(s => s.song_id === song.song_id)) {
                    option.hide();
                }
            });
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to get all collection songs.", data);
    });
}

function remove_from_setlist() {
    let self = $(this);
    let song_id = self.data("song_id");

    if (!song_id) {
        console.error("Unable to get song_id to delete!");
        add_alert("Unable to remove song from setlist", "There was a problem deleting this song. Please refresh the page and try again.", "danger");
        return;
    }

    self.addClass("deleting");
    self.attr("title", "Deleting song... Please wait. If this song doesn't disappear after 30 seconds, please refresh the page.");

    $.ajax({
        url:`/collections/${setlist.collection_id}/setlists/${setlist.setlist_id}/songs/${song_id}`, 
        method: "DELETE",
    })
    .done(function(data) {
        console.log(`Successfully removed song ${song_id}`);
        $(`#songs_container_reorder div[data-id="${song_id}"]`).remove();
        $(`#songs_container a[data-id="${song_id}"]`).hide(500, function() { $(this).remove(); });
        self.hide(500, function() { self.remove(); });
        // refresh_add_song_list();

        $(`#song_select option[value=${song_id}]`).show();
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to remove song from setlist", data);
        self.removeClass("deleting");
        self.removeAttr("title");
    });
}

// Startup function
$(function() {
    // Make reorder list sortable
    $("#songs_container_reorder").sortable();
    $("#songs_container_reorder").disableSelection();

    // Load setlist information
    refresh_setlist();
    refresh_setlist_songs();
});

// #region Edit Setlist
$("#edit_button").click(function() {
    // Populate modal inputs
    $("#setlist_name_input").val(setlist.name);
    $("#setlist_notes_input").val(setlist.notes);
    // Get correct format for the date input
    if (setlist.date) {
        console.log("Setlist UTC Date: " + setlist.date.getUTCDate());
        let year = setlist.date.getUTCFullYear();
        let month = String(setlist.date.getUTCMonth() + 1).padStart(2, '0');
        let day = String(setlist.date.getUTCDate()).padStart(2, '0');
        let date_string = year + "-" + month + "-" + day;
        console.log("Date: " + date_string);
        $("#setlist_date_input").val(date_string);
    }

    $("#edit_setlist_modal").modal("show");
});
$("#save_setlist_button").click(function() {
    save_setlist = true;
    $("#edit_setlist_modal").modal("hide");
});
$("#edit_setlist_modal").on("hidden.bs.modal", function (e) {
    if (save_setlist) {
        $("#edit_setlist_wait").modal("show");
    }
});
$('#edit_setlist_wait').on('shown.bs.modal', function (e) {
    let payload = {
		name: $("#setlist_name_input").val(), 
		date: $("#setlist_date_input").val(), 
		notes: $("#setlist_notes_input").val(),
    };
    if (payload.date.length > 0) {
        payload.date = new Date(payload.date).toISOString();
    } else {
        payload.date = undefined;
    }
    console.log("Updating setlist: " + payload);
    $.ajax({
        method: "PUT",
        url: `/collections/${setlist.collection_id}/setlists/${setlist.setlist_id}`,
        data: JSON.stringify(payload),
        headers: {
            "Content-Type": "application/json"
        }
    })
    .done(function(data) {
		console.log("Edit setlist result:");
        console.log(data);
        add_alert("Changes saved.", "Changes to this setlist have been saved.", "success");
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to save setlist.", data);
    })
    .always(function() {
        $("#edit_setlist_wait").modal("hide");
        refresh_setlist();
        save_setlist = false;
    });
});
// #endregion

// #region Delete setlist
$("#delete_button").click(function() {
    $("#delete_setlist_modal").modal("show");
});
$("#delete_setlist").click(function() {
    delete_setlist = true;
    $("#delete_setlist_modal").modal("hide");
});
$('#delete_setlist_modal').on('hidden.bs.modal', function (e) {
    if (delete_setlist) {
        $("#delete_setlist_wait").modal("show");
    }
});
$('#delete_setlist_wait').on('shown.bs.modal', function (e) {
    $.ajax({
        method: "DELETE",
        url: `/collections/${setlist.collection_id}/setlists/${setlist.setlist_id}`
    })
    .done(function(data) {
        console.log("Setlist delete.");
        console.log(data);
        window.location.replace("/setlists.html?collection_id=" + setlist.collection_id);
    })
    .fail(function(data) {
        $("#delete_setlist_wait").modal("hide");
        alert_ajax_failure("Unable to delete setlist!", data);
    })
    .always(function() {
        delete_setlist = false;
    });
});
// #endregion

// #region Add song
function remove_added_song() {
    let song_id = $(this).data("song_id");
    let song = all_songs.find(song => song.song_id == song_id);
    $(`#song_select option[value='${song_id}']`).show();
    $(this).parent().remove();

    if ($("#song_list").children().length === 0) {
        $("#add_song_modal_button").prop("disabled", true);
    }
}

$("#song_select").change(function() {
    let self = $(this);

    let song_id = self.val();
    let song = all_songs.find(song => song.song_id == song_id);

    let remove_added_song_button = $("<button type='button' class='close' title='Remove song from list of songs to add'>");
    remove_added_song_button.append($("<span aria-hidden='true'>").html("&times;"));
    remove_added_song_button.data("song_id", song.song_id);
    remove_added_song_button.click(remove_added_song);

    $(`#song_select option[value='${song_id}']`).hide();
    self.val(0);
    $("#song_list").append(
        $("<div class='list-group-item'>").data("song_id", song_id).text(song.name).append(remove_added_song_button)
    );

    $("#add_song_modal_button").prop("disabled", false);
});

$("#add_song_modal_button").click(function() {
    add_songs = true;
    $("#setlist_add_song_modal").modal("hide");
});
$("#setlist_add_song_modal").on("hidden.bs.modal", function() {
    if (add_songs) {
        $("#setlist_add_song_wait_modal").modal("show");
        add_songs = false;
    }
});
$("#setlist_add_song_wait_modal").on("shown.bs.modal", function() {
    let payload = $("#song_list div").map(function() {
        return parseInt($(this).data("song_id"), 10);
    }).get();
    console.log(payload);
    console.log(JSON.stringify(payload));
    $.post({
        url: `/collections/${setlist.collection_id}/setlists/${setlist.setlist_id}/songs`,
        data: JSON.stringify(payload),
        contentType: "application/json; charset=utf-8",
    })
    .done(function(data) {
        console.log("Response for adding songs " + payload + " to setlist " + setlist.setlist_id);
        console.log(data);
        $("#song_list").empty();
        $("#add_song_button").prop("disabled", true);
        refresh_setlist_songs();
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to add songs to setlist.", data);
    })
    .always(function() {
        $("#setlist_add_song_wait_modal").modal("hide")
        $("#add_song_modal_button").prop("disabled", true);
    });
});
// #endregion

// #region Reorder songs
$("#reorder_button").click(function() {
    set_mode(modes.REORDER);
});

$("#save_button").click(function() {
    $("#setlist_order_wait_modal").modal("show");
});
$("#setlist_order_wait_modal").on("shown.bs.modal", function() {
    let payload = [];
    $("#songs_container_reorder div").each(function(index, element) { 
        payload.push({
            song_id: $(element).data("song_id"),
            order: index+1, // Avoid zero, and that causes the Go backend to assume its null
        });
    });

    console.log("Updating setlist order:");
    console.log(payload);
    $.ajax({
        method: "PUT",
        url: `/collections/${setlist.collection_id}/setlists/${setlist.setlist_id}/songs`,
        data: JSON.stringify(payload),
        headers: {
            "Content-Type": "application/json"
        }
    })
    .done(function(data) {
		console.log("Edit setlist order result:");
        console.log(data);
        refresh_setlist_songs();
        set_mode(modes.NORMAL);
        add_alert("Changes saved.", "The order of songs in this setlist has been updated.", "success");
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to update setlist order.", data);
    })
    .always(function() {
        $("#setlist_order_wait_modal").modal("hide");
    });
});
// #endregion

// #region Share setlist
$("#share_setlist_modal").on("show.bs.modal", function() {
    $("#share_result").addClass("hidden");
    let visibility = undefined;

    if (setlist.shared) {
        if (setlist.share_code) {
            $("#public_radio").prop("checked", true);
            visibility = "public";
        } else {
            $("#collection_radio").prop("checked", true);
            visibility = "collection";
        }
    } else {
        $("#private_radio").prop("checked", true);
        visibility = "private";
    }
    
    update_share_link(visibility);
});

function update_share_link(visibility) {
    let share_link = get_share_link(visibility)
    if (share_link) {
        $("#share_link_input").val(share_link);
        $("#share_code_form_group").removeClass("hidden");
    } else {
        $("#share_code_form_group").addClass("hidden");
    }
}

$("input[name=visibility_radio]").change(function() {
    $("#save_setlist_visibility_button").prop("disabled", false);
});

function get_share_link(visibility) {
    switch (visibility) {
        case "private":
            return "";
            
        case "collection":
            return window.location.href;

        case "public":
            return `${window.location.origin}/view_setlist.html?code=${setlist.share_code}`;

        default:
            console.error("Unknown visibility value: " + visibility);
            return undefined;
    }
}

$("#save_setlist_visibility_button").click(function() {
    let visibility = $("input[name=visibility_radio]:checked").val();
    
    $("#save_setlist_visibility_button").prop("disabled", true);
    $("#saving").removeClass("hidden");

    console.log("Updating setlist visibility to: " + visibility);
    $.ajax({
        method: "PUT",
        url: `/collections/${setlist.collection_id}/setlists/${setlist.setlist_id}/visibility`,
        data: visibility,
        headers: {
            "Content-Type": "application/json"
        }
    })
    .done(function(data) {
		console.log("Edit setlist visibility result:");
        console.log(data);

        $("#share_result").removeClass("hidden");

        switch (visibility) {
            case "private":
                $("#private_result").removeClass("hidden");
                $("#collection_result").addClass("hidden");
                $("#public_result").addClass("hidden");
                break;
                
            case "collection":
                $("#private_result").addClass("hidden");
                $("#collection_result").removeClass("hidden");
                $("#public_result").addClass("hidden");
                break;

            case "public":
                $("#private_result").addClass("hidden");
                $("#collection_result").addClass("hidden");
                $("#public_result").removeClass("hidden");
                setlist.share_code = data;
                break;
        }
        
        update_share_link(visibility);

        // $("#share_result").text("Setlist visibility has been updated!").removeClass("hidden");
        // add_alert("Changes saved.", "Changes to this setlist have been saved.", "success");
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to update setlist visibility.", data);
        $("#share_setlist_modal").modal("hide");
    })
    .always(function() {
        $("#saving").addClass("hidden");
        refresh_setlist();
    });
});

$("#copy_link_button").click(function() {
    navigator.clipboard.writeText($("#share_link_input").val()).then(function() {
        // Clipboard successfully set
        $("#copy_link_success").fadeIn(100, function() {
            $("#copy_link_success").fadeOut(5000);
        })
        console.log("Copied link to clipboard");
    }, function() {
        // Clipboard write failed, try using execCommand
        console.warn("Clipboard API failed, trying execCommand");
        var copyText = document.querySelector("#input");
        $("#share_link_input").select();
        document.execCommand("copy");
        console.log("Copied link to clipboard with execCommand API");
    });
});
// #endregion

$("#remove_button").click(function() {
    set_mode(modes.REMOVE);
});

$("#cancel_button, #back_button").click(function() {
    set_mode(modes.NORMAL);
});

function set_mode(new_mode) {
    mode = new_mode;

    if (mode === modes.NORMAL) {
        $("#songs_container").removeClass("hidden");
        $("#songs_container_delete").addClass("hidden");
        $("#songs_container_reorder").addClass("hidden");
    
        $("#add_button").removeClass("hidden");
        $("#reorder_button").removeClass("hidden");
        $("#remove_button").removeClass("hidden");
        $("#save_button").addClass("hidden");
        $("#cancel_button").addClass("hidden");
        $("#back_button").addClass("hidden");
    
        $("#remove_alert").hide(500);
        $("#reorder_help").hide(500);
    } else if (mode === modes.REMOVE) {
        $("#songs_container").addClass("hidden");
        $("#songs_container_delete").removeClass("hidden");
    
        $("#add_button").addClass("hidden");
        $("#reorder_button").addClass("hidden");
        $("#remove_button").addClass("hidden");
        $("#back_button").removeClass("hidden");
    
        $("#remove_alert").show(500);
    } else if (mode === modes.REORDER) {
        $("#songs_container").addClass("hidden");
        $("#songs_container_reorder").removeClass("hidden");
    
        $("#add_button").addClass("hidden");
        $("#reorder_button").addClass("hidden");
        $("#remove_button").addClass("hidden");
        $("#save_button").removeClass("hidden");
        $("#cancel_button").removeClass("hidden");
    
        $("#reorder_help").show(500);
    }
}
