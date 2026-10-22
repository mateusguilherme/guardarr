import { artistReleases } from "#lib/api.ts";
import type { PageLoad } from "./$types";

// The releases come back unawaited, so the page opens on its skeleton right
// away instead of holding the navigation until MusicBrainz answers.
export const load: PageLoad = ({ params, fetch }) => {
  const releases = artistReleases(params.id, fetch);

  // The page shows a failure through {#await}; until it renders, this
  // keeps the browser from reporting the rejection as unhandled.
  releases.catch(() => {});

  return { releases };
};
