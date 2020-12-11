"use strict";

import { add_alert, alert_ajax_failure, get_youtube_video_id, substitute_URLs } from "./utilities.js";

let url = new URL(window.location.href);
let collection_id = parseInt(url.searchParams.get("collection_id"), 10);
let query = url.searchParams.get("query");
let tags = [];

// Show navbar options
$("#navbar_dashboard").removeClass("hidden");
$("#collection_link").attr("href", "/collection.html?collection_id=" + collection_id);

function tag_button_clicked(e) {
    let tag_id = $(this).data("tag_id");
    
    $(this).toggleClass("btn-light");
    $(this).toggleClass("btn-dark");
}

$(function() {
    // Populate 
    if (query) {
        $("#keywords_include_input").val(query);
    }

	// Get collection tags
    $.get(`/collections/${collection_id}/tags`)
    .done(function(data) {
		tags = data;
		
        console.log("All tags in this collection:");
        console.log(tags);

        // Add tags to option list
        $("#tag_list").empty();
        tags.forEach(tag => {
            let button = $("<button type='button' class='btn btn-light'>")
                            .text(tag.name)
                            .data("tag_id", tag.tag_id)
                            .click(tag_button_clicked);
            $("#tag_list").append(button);
        });
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to get collection tags!", data);
	});
	
	$("#select_all_button").click(function() {
		$("#tag_list button").each(function() {
			$(this).addClass("btn-dark");
			$(this).removeClass("btn-light");
		});
	});
	
	$("#select_none_button").click(function() {
		$("#tag_list button").each(function() {
			$(this).addClass("btn-light");
			$(this).removeClass("btn-dark");
		});
	});
});

$("#search_button").click(function() {
    let payload = {
        tags: [],
        before: null, 
        after: null,
        include: $("#keywords_include_input").val().match(/\S+/g) || [],
        exclude: $("#keywords_exclude_input").val().match(/\S+/g) || [],
        collection_id: collection_id,
    }

    let before = $("#performed_before_input").val();
    let after = $("#performed_after_input").val();

    if (before.length > 0) {
        payload.before = new Date(before).toISOString();
    }
    
    if (after.length > 0) {
        payload.after = new Date(after).toISOString();
    }

    $("#tag_list .btn-dark").each(function() { payload.tags.push($(this).data("tag_id")); });
    
    console.log("Advanced search request:");
    console.log(payload);
    $.post(`/collections/${collection_id}/search`, JSON.stringify(payload))
    .done(function(data) {
        console.log("Response from search:");
        console.log(data);

        $("#search_results").empty();

        if (data.length == 0) {
            $("#search_results").append($("<div>").text("No results."));
        } else {
            data.forEach(song => {
                console.log("Hello from " + song.song_name);
                let element = $("<a>")
                .attr("href", `song.html?collection_id=${collection_id}&song_id=${song.song_id}`)
                .attr("target", "_blank")
                .addClass("list-group-item list-group-item-action")
                .text(song.song_name);
    
                $("#search_results").append(element);
            });
        }

        $("#search_results_container").removeClass("hidden");
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to search.", data);
    });
});