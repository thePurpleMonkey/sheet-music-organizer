import { create_alert } from "./utilities.js";

const MESSAGE = `
Sheet Music Organizer has a new home!
Update your bookmarks, because our new website address is
<a href="https://sheetmusicorganizer.com">sheetmusicorganizer.com</a>.
Thank you for using Sheet Music Organizer!
`;

function dismiss_new_domain_message() {
	try {
		window.localStorage.setItem("new_domain_message_dismissed", "true");
	}  catch (err) {
        console.log("Unable to set localStorage variable 'new_domain_message_dismissed'");
        console.log(err);
	}
}

$(function() {
    // Check for 
	let new_domain_message_dismissed = false;
    try {
        new_domain_message_dismissed = window.localStorage.getItem("new_domain_message_dismissed");
        console.log("new_domain_message_dismissed = " + new_domain_message_dismissed);

		if (new_domain_message_dismissed !== "true") {
			let alert = create_alert("New domain, who dis?", MESSAGE, "info")
			alert.find(".close").click(dismiss_new_domain_message);
			$("#alerts").append(alert);
		}
    } catch (err) {
        console.log("Unable to retrieve localStorage variable 'new_domain_message_dismissed'");
        console.log(err);
    }
})