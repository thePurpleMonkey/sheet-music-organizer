"use strict";

import { add_alert, get_session_alert } from "/js/utilities.js";

// Check for any alerts
let alert = get_session_alert();
if (alert) {
	add_alert(alert.title, alert.message, alert.style);
}