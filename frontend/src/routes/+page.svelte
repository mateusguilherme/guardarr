<script lang="ts">
  import { resolve } from "$app/paths";
  import type { Artist } from "#lib/api.ts";
  import { getAppState } from "#lib/app-state.svelte.ts";
  import Notice from "#lib/Notice.svelte";

  const app = getAppState();

  let searching = $derived(app.search.status === "searching");

  function submit(event: SubmitEvent) {
    event.preventDefault();
    app.runSearch();
  }

  // fold lowercases and strips accents, so "jose" matches "José".
  function fold(s: string) {
    return s
      .normalize("NFD")
      .replace(/[̀-ͯ]/g, "")
      .toLowerCase();
  }

  // The home list narrows to the typed term as the user types, before
  // any search is submitted.
  let filteredFollowed = $derived.by(() => {
    const q = fold(app.term.trim());
    const all = [...app.followed.values()];

    if (!q) {
      return all;
    }

    return all.filter(
      (artist) =>
        fold(artist.name).includes(q) ||
        fold(artist.disambiguation ?? "").includes(q),
    );
  });

  // Emptying the field by any means (escape, backspace, delete…) returns
  // to the followed artists, discarding any search still in flight.
  $effect(() => {
    if (app.term.trim() === "" && app.search.status !== "idle") {
      app.resetSearch();
    }
  });

  function keydown(event: KeyboardEvent) {
    if (event.key !== "Escape") {
      return;
    }

    // First escape clears the text (the effect above restores the home);
    // with nothing left to clear, a second one leaves the field.
    if (app.term === "") {
      (event.currentTarget as HTMLInputElement).blur();
      return;
    }

    app.term = "";
  }
</script>

{#snippet artistList(artists: Artist[])}
  <ul>
    {#each artists as artist, i (artist.id)}
      <li style="--i: {i}">
        <a class="open-artist" href={resolve("/artist/[id]", { id: artist.id })}>
          <span class="avatar" aria-hidden="true">{artist.name.charAt(0).toUpperCase()}</span>
          <div class="who">
            <span class="name">{artist.name}</span>
            {#if artist.disambiguation}
              <span class="disambiguation">{artist.disambiguation}</span>
            {/if}
          </div>
        </a>
        {#if app.followed.has(artist.id)}
          <button
            class="following"
            onclick={() => app.unfollow(artist.id)}
            disabled={app.pending.has(artist.id)}
            aria-label="Unfollow {artist.name}"
          >
            <span class="when-idle">
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <path
                  d="M5 12.5 10 17.5 19 7"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2.5"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
              Following
            </span>
            <span class="when-hover">Unfollow</span>
          </button>
        {:else}
          <button
            class="follow"
            onclick={() => app.follow(artist)}
            disabled={app.pending.has(artist.id)}
          >
            Follow
          </button>
        {/if}
      </li>
    {/each}
  </ul>
{/snippet}

<header class="hero">
  <h1>
    <svg class="mark" viewBox="0 0 24 24" aria-hidden="true">
      <circle cx="10" cy="10" r="7.25" fill="none" stroke="currentColor" stroke-width="2.5" />
      <circle cx="10" cy="10" r="4" fill="none" stroke="currentColor" stroke-width="1" opacity="0.5" />
      <circle cx="10" cy="10" r="1.5" fill="currentColor" />
      <path d="M15.5 15.5 21 21" stroke="currentColor" stroke-width="2.75" stroke-linecap="round" />
    </svg>
    <span>guard<span class="arr">arr</span></span>
  </h1>
  <p class="tagline">
    <i>guardar</i> <span class="ipa">/ɡwaʁˈdaʁ/</span> — Portuguese for “to
    keep, to watch over”
  </p>
</header>

<form class="search" onsubmit={submit}>
  <svg class="search-icon" viewBox="0 0 24 24" aria-hidden="true">
    <circle cx="11" cy="11" r="7" fill="none" stroke="currentColor" stroke-width="2" />
    <path d="M16.5 16.5 21 21" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
  </svg>
  <input bind:value={app.term} placeholder="Search an artist…" onkeydown={keydown} />
  <button disabled={searching}>
    {#if searching}<span class="spinner" aria-hidden="true"></span>{/if}
    {searching ? "Searching…" : "Search"}
  </button>
</form>

{#if app.search.status === "searching"}
  <div class="skeletons" aria-hidden="true">
    {#each { length: 4 }, i}
      <div class="skeleton" style="--i: {i}">
        <div class="skeleton-avatar"></div>
        <div class="skeleton-lines">
          <div class="skeleton-line"></div>
          <div class="skeleton-line short"></div>
        </div>
      </div>
    {/each}
  </div>
{:else if app.search.status === "error"}
  <Notice error message={app.search.message} />
{:else if app.search.status === "done" && app.search.artists.length > 0}
  <section class="results">
    <p class="count">
      {app.search.artists.length}
      {app.search.artists.length === 1 ? "artist" : "artists"}
    </p>
    {@render artistList(app.search.artists)}
  </section>
{:else if app.search.status === "done"}
  <Notice message="No artists found." hint="Try a different spelling or another name.">
    {#snippet icon()}
      <circle cx="11" cy="11" r="7" fill="none" stroke="currentColor" stroke-width="2" />
      <path d="M16.5 16.5 21 21" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
      <path d="M8.5 11h5" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
    {/snippet}
  </Notice>
{:else if app.search.status === "idle" && app.followed.size > 0}
  <section class="results">
    <p class="count">
      {#if app.term.trim()}
        {filteredFollowed.length} of {app.followed.size} followed
      {:else}
        {app.followed.size}
        {app.followed.size === 1 ? "artist followed" : "artists followed"}
      {/if}
    </p>
    {#if filteredFollowed.length > 0}
      {@render artistList(filteredFollowed)}
    {:else}
      <p class="no-match">
        No followed artists match — press Enter to search MusicBrainz.
      </p>
    {/if}
  </section>
{/if}
