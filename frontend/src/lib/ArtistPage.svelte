<script lang="ts">
  import {
    artistReleases,
    followedArtists,
    lookupArtist,
    type Release,
  } from "./api";

  let { id, name }: { id: string; name?: string } = $props();

  let fetchedName = $state<string>();
  let displayName = $derived(name ?? fetchedName);

  // A deep link carries only the id. The name is then recovered from the
  // followed artists saved locally, or failing that from MusicBrainz.
  async function resolveName(id: string): Promise<string> {
    try {
      const saved = (await followedArtists()).find((artist) => artist.id === id);

      if (saved) {
        return saved.name;
      }
    } catch {
      // not fatal; MusicBrainz can still answer
    }

    return (await lookupArtist(id)).name;
  }

  $effect(() => {
    if (name !== undefined) {
      return;
    }

    let stale = false;

    fetchedName = undefined;

    resolveName(id).then(
      (resolved) => {
        if (!stale) {
          fetchedName = resolved;
        }
      },
      () => {
        // the releases still load; only the heading stays blank
      },
    );

    return () => {
      stale = true;
    };
  });

  type Releases =
    | { status: "loading" }
    | { status: "done"; releases: Release[] }
    | { status: "error"; message: string };

  let releases = $state<Releases>({ status: "loading" });

  // Reloads whenever the artist changes; a response from a previous artist
  // is discarded instead of landing on the new page.
  $effect(() => {
    let stale = false;

    releases = { status: "loading" };

    artistReleases(id).then(
      (list) => {
        if (!stale) {
          releases = { status: "done", releases: list };
        }
      },
      (err) => {
        if (!stale) {
          releases = {
            status: "error",
            message: err instanceof Error ? err.message : String(err),
          };
        }
      },
    );

    return () => {
      stale = true;
    };
  });

  // Covers come straight from the Cover Art Archive; a release without one
  // keeps the vinyl placeholder behind the broken image.
  function hideBrokenCover(event: Event) {
    (event.currentTarget as HTMLImageElement).hidden = true;
  }
</script>

<header class="artist-header">
  <a class="back" href="#/" aria-label="Back to your artists">
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path
        d="M14.5 6 8.5 12l6 6"
        fill="none"
        stroke="currentColor"
        stroke-width="2.5"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>
  </a>
  <span class="avatar" aria-hidden="true">
    {displayName?.charAt(0).toUpperCase() ?? ""}
  </span>
  {#if displayName}
    <h2>{displayName}</h2>
  {:else}
    <span class="skeleton-line" aria-hidden="true"></span>
  {/if}
</header>

{#if releases.status === "loading"}
  <div class="release-skeletons" aria-hidden="true">
    {#each { length: 6 } as _, i}
      <div class="skeleton-release" style="--i: {i}">
        <div class="skeleton-cover"></div>
        <div class="skeleton-line"></div>
        <div class="skeleton-line short"></div>
      </div>
    {/each}
  </div>
{:else if releases.status === "error"}
  <div class="notice" role="alert">
    <svg class="notice-icon error-icon" viewBox="0 0 24 24" aria-hidden="true">
      <circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" stroke-width="2" />
      <path d="M12 7.5v5.5" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
      <circle cx="12" cy="16.5" r="1.25" fill="currentColor" />
    </svg>
    <p class="error">{releases.message}</p>
    <p class="hint">MusicBrainz may be rate-limiting; try again in a moment.</p>
  </div>
{:else if releases.releases.length > 0}
  <section class="results">
    <p class="count">
      {releases.releases.length} official
      {releases.releases.length === 1 ? "release" : "releases"}
    </p>
    <ul class="releases">
      {#each releases.releases as release, i (release.id)}
        <li style="--i: {i}">
          <div class="cover">
            <svg class="cover-fallback" viewBox="0 0 24 24" aria-hidden="true">
              <circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" stroke-width="1.5" />
              <circle cx="12" cy="12" r="4" fill="none" stroke="currentColor" stroke-width="1" opacity="0.5" />
              <circle cx="12" cy="12" r="1.25" fill="currentColor" />
            </svg>
            <img
              src="https://coverartarchive.org/release-group/{release.id}/front-250"
              alt=""
              loading="lazy"
              onerror={hideBrokenCover}
            />
          </div>
          <span class="release-title">{release.title}</span>
          {#if release.firstReleaseDate}
            <span class="release-year">{release.firstReleaseDate.slice(0, 4)}</span>
          {/if}
        </li>
      {/each}
    </ul>
  </section>
{:else}
  <div class="notice">
    <svg class="notice-icon" viewBox="0 0 24 24" aria-hidden="true">
      <circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" stroke-width="2" />
      <circle cx="12" cy="12" r="4" fill="none" stroke="currentColor" stroke-width="1" opacity="0.5" />
      <circle cx="12" cy="12" r="1.25" fill="currentColor" />
    </svg>
    <p class="empty">No official releases.</p>
    <p class="hint">Only official albums show here for now.</p>
  </div>
{/if}
