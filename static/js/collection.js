"use strict";

import { add_alert, getUrlParameter, alert_ajax_failure, get_session_alert, dates } from "./utilities.js";
import "./modernizr-touch.js";

let add_song = false;
let add_tag = false;
let edit_collection = false;
let delete_collection = false;

let tutorial = false;

let collection = {
    // Parse collection ID from URL parameter
    id: getUrlParameter("collection_id"),

    // These attributes get set after an AJAX call to server
    name: undefined,
    description: undefined,
};

// Object for storing collection settings
let settingsKey = `collection_${collection.id}`;
let settings = undefined;

// Replace links
$("#members_link").attr("href", "/members.html?collection_id=" + collection.id);
$("#filter_link").attr("href", "/advanced_search.html?collection_id=" + collection.id);
$("#setlists_link").attr("href", "/setlists.html?collection_id=" + collection.id);
$("#advanced_search_dropdown_link").attr("href", "/advanced_search.html?collection_id=" + collection.id);

// Show options in navbar
$("#navbar_options").show();
$("#navbar_members").show();
$("#leave_button").removeClass("hidden");
$("#search_form").removeClass("hidden");
$("#navbar_settings").removeClass("hidden");
$("#navbar_setlists").removeClass("hidden");

// Get collection info when document becomes ready
$(function() {
	// Enable form validation
    $("#add_song_form").validate();
    $("#add_tag_form").validate();

    // Load dashboard settings
    reloadSettings();

    console.log("Touch enabled: " + Modernizr.touchevents)
    if (Modernizr.touchevents) {
        $(".clicking").text("tapping");
        $(".click").text("tap");
        $(".Click").text("Tap");
    }
    
    // Get collection data
    $.get(`/collections/${collection.id}`)
    .done(function(data) {
        console.log(data);
        collection.name = data.name;
        collection.description = data.description;
        
        if (!data.admin) {
            $("#edit_button").hide();
            $("#delete_button").hide();
            $("#members_divider").hide();
        }

        $("#page_header").text(collection.name);
        $("#collection_name").val(collection.name);
        $("#collection_description").val(collection.description);

        if (collection.description) {
            $("#description").text(collection.description);
        } else {
            $("#description_row").hide();
        }
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to get collection information!", data);
    });

	// Check for any alerts
	let alert = get_session_alert();
	if (alert) {
		add_alert(alert.title, alert.message, alert.style);
    }
    
    // Show tutorial when dashboard has fully loaded
    $.when(reloadSongs(), reloadTags()).then(function() {
        console.log("Showing tutorial");
        initialize_tutorial();
    });
});

function name_compare(item1, item2) {
    return item1.name.localeCompare(item2.name);
}

function added_compare(item1, item2) {
    return dates.compare(item1.date_added, item2.date_added)
}

function reloadSongs() {
    $("#songs").empty();
    $("#songs").append('<a class="list-group-item">Loading songs, please wait...</a>');
    let payload = undefined;
    if (settings.hidden_tags.length > 0) {
        payload = { exclude_tags: JSON.stringify(settings.hidden_tags) };
    }

    let result = $.Deferred();

    $.get(`/collections/${collection.id}/songs`, payload)
    .done(function(data) {
        console.log("Get songs result:");
        console.log(data);
        $("#songs").empty();

        if (settings.song_sort === "name") {
            data.sort(name_compare);
        } else if (settings.song_sort === "date_added") {
            data.sort(added_compare);
        } else {
            console.log("Unknown song sorting function: " + settings.song_sort);
        }

        data.forEach(song => {
            let a = $("<a>");
            a.addClass("list-group-item");
            a.addClass("list-group-item-action");
            a.attr("href", `/song.html?song_id=${encodeURIComponent(song.song_id)}&collection_id=${collection.id}`);
            a.text(song.name);
            $("#songs").append(a);
        });

        if (tutorial) { update_tutorial() }

        result.resolve(data);
    })
    .fail(function(data) {
        if (data.status === 403) {
            window.location.replace("/404.html");
        }

        alert_ajax_failure("Unable to get songs!", data);
        result.reject(data);
    })
    .always(function() {
        $("#loading").remove();
    });

    return result.promise();
};

function reloadTags() {
    $("#tags").empty();
    $("#hide_tag_list").empty();

    $("#tags").append('<a class="list-group-item">Loading tags, please wait...</a>');

    let result = $.Deferred();
    
    $.get(`/collections/${collection.id}/tags`)
    .done(function(data) {
        console.log("Get tags result:");
        console.log(data);
        $("#tags").empty();

        if (settings.tag_sort === "name") {
            data.sort(name_compare);
        } else if (settings.tag_sort === "date_added") {
            // data.sort(added_compare);
            data.sort((a, b) => a.tag_id - b.tag_id);
        } else {
            console.log("Unknown tag sorting function: " + settings.tag_sort);
        }

        data.forEach(tag => {
            // Do not add tag to dashboard if hidden
            if (!settings.hidden_tags.includes(tag.tag_id)) {
                let a = $("<a>");
                a.addClass("list-group-item");
                a.addClass("list-group-item-action");
                a.attr("href", `/tag.html?tag_id=${encodeURIComponent(tag.tag_id)}&collection_id=${collection.id}`);
                a.text(tag.name);
                $("#tags").append(a);
            }
            
            // Add tag to collection settings
            let button = $("<button type='button' class='btn'>")
                            .text(tag.name)
                            .data("tag_id", tag.tag_id)
                            .click(function() {
                                $(this).toggleClass("btn-light");
                                $(this).toggleClass("btn-dark");
                            });
            if (settings.hidden_tags.includes(tag.tag_id)) {
                button.addClass("btn-dark");
            } else {
                button.addClass("btn-light");
            }
            $("#hide_tag_list").append(button);
        });

        if (tutorial) { update_tutorial() }

        result.resolve(data);
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to get tags!", data);
        result.reject(data);
    })
    .always(function() {
        $("#loading").remove();
    });

    return result.promise();
};

function reloadSettings() {
    // Get settings from local storage
    console.log("Loading settings from key: " + settingsKey);

    let settings_string = localStorage.getItem(settingsKey);
    if (settings_string === null) {
        console.log("No settings found. Initializing settings object");
        settings = {
            hidden_tags: [],
            song_sort: "name",
            tag_sort: "name",
        };
        saveSettings();
    } else {
        try {
            console.log("Settings string: " + settings_string);
            settings = JSON.parse(settings_string);
        } catch (err) {
            console.error("Unable to load settings!");
            console.error(err);
            add_alert("Unable to load settings", "There was a problem loading the settings for this collection. Please refresh the page and try again.", "warning");
            return
        }
    }
    
    console.log("Settings:");
    console.log(settings);

    // Update song sort settings UI
    if (settings.song_sort === "name") {
        $("#song_sort_name").prop("checked", true);
    } else if (settings.song_sort === "date_added") {
        $("#song_sort_added").prop("checked", true);
    } else {
        console.warn("Unknown song sort name: " + settings.song_sort);
    }

    // Update tag sort settings UI
    if (settings.tag_sort === "name") {
        $("#tag_sort_name").prop("checked", true);
    } else if (settings.tag_sort === "date_added") {
        $("#tag_sort_added").prop("checked", true);
    } else {
        console.warn("Unknown tag sort name: " + settings.tag_sort);
    }

    // Update hidden tags settings UI
    $("#hide_tag_list").children().each(function() {
        let tag = $(this);
        let tag_id = tag.data("tag_id");
        tag.removeClass("btn-light btn-dark");
        if (settings.hidden_tags.includes(tag_id)) {
            tag.addClass("btn-dark");
        } else {
            tag.addClass("btn-light");
        }
    });
}

function saveSettings() {
    let settings_string = JSON.stringify(settings);
    try {
        console.log("Saving settings to: " + settingsKey);
        localStorage.setItem(settingsKey, settings_string);
        console.log("Saved settings:");
        console.log(settings);
    } catch (err) {
        console.error("Unable to save settings to local storage!");
        console.error(err);
        add_alert("Unable to save settings", "The settings for this collection were unable to be saved. Please refresh the page and try again.", "danger");
    }
}

// #region Add song
function tag_button_clicked(e) {
    $(this).toggleClass("btn-light");
    $(this).toggleClass("btn-dark");
}

$("#add_song_modal").on("show.bs.modal", function() {
    $.get(`/collections/${collection.id}/tags`)
    .done(function(data) {

        console.log("All tags in this collection:");
        console.log(data);
        
        // Add tags to option list
        $("#tags_list").empty();
        data.forEach(item => {
            let button = $("<button type='button' class='btn btn-light'>")
                            .text(item.name)
                            .data("tag_id", item.tag_id)
                            .click(tag_button_clicked);
            $("#tags_list").append(button);
        });
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to get your tags!", data);
    });
});

$("#add_song_modal_button").click(function() {
    if ($("#add_song_form").valid()) {
        add_song = true;
        $("#add_song_modal").modal("hide");
    }
});
$('#add_song_modal').on('hidden.bs.modal', function (e) {
    if (add_song) {
        $("#song_wait").modal("show");
    }
});

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
        console.log("Successfully added song! API response:");
        console.log(data);

        // Clear form fields
        $("#name").val("");
        $("#artist").val("");
        $("#location").val("");
        $("#last_performed").val("");
        $("#notes").val("");

        add_song_tags(data.song_id);
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to add song!", data);
        $("#song_wait").modal("hide");
    })
    .always(function() {
        add_song = false;
        reloadSongs();
    });
});
function add_song_tags(song_id) {
    let requests = [];

    $("#tags_list > .btn-dark").each(function() {
        let tag_id = $(this).data("tag_id");
        console.log(`Adding tag ID ${tag_id} to song ID ${song_id}`);
        let payload = JSON.stringify({song_id: song_id, tag_id: parseInt(tag_id, 10)});

        requests.push(
            $.post(`/collections/${collection.id}/songs/${song_id}/tags`, payload)
            .done(function(data) {
                console.log("Response for adding tag " + tag_id + " to song " + song_id);
                console.log(data);
            })
            .fail(function(data) {
                alert_ajax_failure("Unable to add tag to new song.", data);
            })
        );
    });

    $.when(requests)
    .done(function() {
        console.log(`Successfully added tags to song ${song_id}`);
        add_alert("Song added!", "The song was successfully added to your collection.", "success");
    })
    .always(function() {
        $("#song_wait").modal("hide");
    });
}
//#endregion

// Attach to navbar buttons
$("#edit_button").click(function() {
    $("#edit_collection_modal").modal("show");
});
$("#delete_button").click(function() {
    $("#delete_collection_modal").modal("show");
});

// #region Save changes to collection
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
//#endregion

// #region Delete collection
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
//#endregion

// #region Add Tag
$("#add_tag_modal_button").click(function() {
    if ($("#add_tag_form").valid()) {
        add_tag = true
        $("#add_tag_modal").modal("hide");
    }
});
$('#add_tag_modal').on('hidden.bs.modal', function (e) {
    if (add_tag) {
        add_tag = false;
        $("#tag_wait").modal("show");
    }
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
        console.log("Add tag response:");
        console.log(data);

        // Clear add tag fields
        $("#tag_name").val("");
        $("#tag_description").val("");

        // Display success message
        add_alert("Tag created!", "The tag was successfully created. You may now start tagging your songs with it.", "success");
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to add tag!", data);
    })
    .always(function() {
        $("#tag_wait").modal("hide");
        reloadTags();
        $("#add_tag").val("");
        $("#tag_description").val("");
    });
});
// #endregion

// #region Save Settings
$("#settings_save_button").click(function() {
    settings.hidden_tags = [];
    $("#hide_tag_list .btn-dark").each(function() {
        settings.hidden_tags.push($(this).data("tag_id"));
    });

    // Save song sort order
    settings.song_sort = $('input[name=song_sort_order]:checked').val();
    settings.tag_sort = $('input[name=tag_sort_order]:checked').val();
    saveSettings();

    reloadSongs();
    reloadTags();

    $("#settings_modal").modal("hide");
});

$("#settings_modal").on("hidden.bs.modal", function() {
    reloadSettings();
});
// #endregion

// #region Tutorial
function hide_tutorial() {
	console.log("Hide tutorial clicked.");
	try {
		window.localStorage.setItem("show_tutorial", false);
	} catch (err) {
		console.warn("Unable to hide tutorial");
		console.warn(err);
	} finally {
		$(".tutorial").hide(500);
	}
}

function initialize_tutorial() {
	tutorial = window.localStorage.getItem("show_tutorial");
	console.log("Tutorial: " + tutorial);
	if (tutorial != "false") {
        // Show tutorials
        $("#add_song_tutorial_alert").removeClass("hidden");
        $("#add_tag_tutorial_alert").removeClass("hidden");

        update_tutorial();

		$(".hide_tutorial").click(hide_tutorial);
	}
}

// Updates the tutorial alert on the dashboard
function update_tutorial() {
    console.log("Updating tutorial");
    console.log("Tags: " + $("#tags").children().length);
    console.log("Songs: " + $("#songs").children().length);
    if ($("#tags").children().length == 0) {
        $("#dashboard_add_tag_tutorial_alert").removeClass("hidden");
        $("#dashboard_add_song_tutorial_alert").addClass("hidden");
        $("#dashboard_complete_tutorial_alert").addClass("hidden");
    } else if ($("#songs").children().length == 0) {
        $("#dashboard_add_song_tutorial_alert").removeClass("hidden");
        $("#dashboard_add_tag_tutorial_alert").addClass("hidden");
        $("#dashboard_complete_tutorial_alert").addClass("hidden");
    } else {
        $("#dashboard_add_song_tutorial_alert").addClass("hidden");
        $("#dashboard_add_tag_tutorial_alert").addClass("hidden");
        $("#dashboard_complete_tutorial_alert").removeClass("hidden");
    }
}
// #endregion
