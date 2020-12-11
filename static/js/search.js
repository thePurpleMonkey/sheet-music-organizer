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

    // Construct link for advanced search page
    let search_link = new URL(window.location.origin);
    search_link.pathname = "/advanced_search.html";
    search_link.searchParams.set("collection_id", collection_id);
    search_link.searchParams.set("query", query);
    console.log("Advanced Search href: " + search_link);
	$("#advanced_search_link").attr("href", search_link.href);
    
    console.log("Query: " + query);
    // let payload = { query: encodeURIComponent(query) };
    let payload = {query: query};
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