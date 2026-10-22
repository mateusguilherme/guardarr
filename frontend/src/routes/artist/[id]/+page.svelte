<script lang="ts">
  import { resolve } from "$app/paths";
  import { lookupArtist } from "#lib/api.ts";
  import { getAppState } from "#lib/app-state.svelte.ts";
  import type { PageProps } from "./$types";

  let { data, params }: PageProps = $props();

  const app = getAppState();

  let name = $derived(app.knownArtist(params.id)?.name);
  let fetchedName = $state<string>();
  let displayName = $derived(name ?? fetchedName);

  // A deep link carries only the id. Once the followed artists load, the
  // name of a followed one is known; only an artist missing from that list
  // is looked up on MusicBrainz.
  $effect(() => {
    if (name !== undefined || !app.followedLoaded) {
      return;
    }

    let stale = false;

    fetchedName = undefined;

    lookupArtist(params.id).then(
      (artist) => {
        if (!stale) {
          fetchedName = artist.name;
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

  // Covers come straight from the Cover Art Archive; a release without one
  // keeps the vinyl placeholder behind the broken image.
  function hideBrokenCover(event: Event) {
    (event.currentTarget as HTMLImageElement).hidden = true;
  }
</script>

<header class="artist-header">
  <a class="back" href={resolve("/")} aria-label="Back to your artists">
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

{#await data.releases}
  <div class="release-skeletons" aria-hidden="true">
    {#each { length: 6 }, i}
      <div class="skeleton-release" style="--i: {i}">
        <div class="skeleton-cover"></div>
        <div class="skeleton-line"></div>
        <div class="skeleton-line short"></div>
      </div>
    {/each}
  </div>
{:then releases}
  {#if releases.length > 0}
    <section class="results">
      <p class="count">
        {releases.length} official
        {releases.length === 1 ? "release" : "releases"}
      </p>
      <ul class="releases">
        {#each releases as release, i (release.id)}
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
{:catch err}
  <div class="notice" role="alert">
    <svg class="notice-icon error-icon" viewBox="0 0 24 24" aria-hidden="true">
      <circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" stroke-width="2" />
      <path d="M12 7.5v5.5" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
      <circle cx="12" cy="16.5" r="1.25" fill="currentColor" />
    </svg>
    <p class="error">{err instanceof Error ? err.message : String(err)}</p>
    <p class="hint">MusicBrainz may be rate-limiting; try again in a moment.</p>
  </div>
{/await}
