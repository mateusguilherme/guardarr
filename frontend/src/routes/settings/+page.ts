import { settings } from "#lib/api.ts";
import type { PageLoad } from "./$types";

// The settings come from guardarr's own database, so the page waits for
// them. slskd itself, which can take seconds to answer, is checked once the
// page is up.
export const load: PageLoad = ({ fetch }) => settings(fetch);
