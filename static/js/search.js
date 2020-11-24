"use strict";

import { getUrlParameter, alert_ajax_failure } from "./utilities.js";

let collection_id = getUrlParameter("collection_id");
let query = new URL(window.location.href).searchParams.get("query");

// Show navbar options
$("#navbar_dashboard").removeClass("hidden");
$("#search_form").removeClass("hidden");
$("#search_box").val(query);

$(function() {
    // Replace link for collection
	$("#collection_link").attr("href", "/collection.html?collection_id=" + collection_id);
	
	let payload = { query: encodeURIComponent(query) }
    $.get(`/collections/${collection_id}/search`, payload)
    .done(function(data) {
		console.log("Search response:");
        console.log(data);

		$("#search_results").empty();

        data.forEach(result => {
            let element = $("<a>")
            .attr("href", `song.html?collection_id=${collection_id}&song_id=${result.song_id}`)
            .addClass("list-group-item list-group-item-action")
			.text(result.song_name);

            $("#search_results").append(element);
        });
    })
    .fail(function(data) {
        alert_ajax_failure("Unable to search.", data);
    });
});