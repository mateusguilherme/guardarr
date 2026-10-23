import { defineParams } from "@sveltejs/kit/params";

// An artist's address carries its MusicBrainz id, a lowercase UUID. Any
// other text matches no page, so the app shows its not-found page instead
// of asking MusicBrainz about it.
export const params = defineParams({
  mbid: (param) =>
    /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/.test(param) ? param : undefined,
});
