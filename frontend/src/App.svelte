<script lang="ts">
  import { SvelteSet } from "svelte/reactivity";
  import {
    followArtist,
    followedArtists,
    searchArtists,
    type Artist,
  } from "./lib/api";

  type Search =
    | { status: "idle" }
    | { status: "searching" }
    | { status: "done"; artists: Artist[] }
    | { status: "error"; message: string };

  let term = $state("");
  let search = $state<Search>({ status: "idle" });
  let searching = $derived(search.status === "searching");

  const followed = new SvelteSet<string>();
  const pending = new SvelteSet<string>();

  // Failing to load the list only means every button starts as "Follow".
  followedArtists().then(
    (ids) => ids.forEach((id) => followed.add(id)),
    () => {},
  );

  async function follow(id: string) {
    if (followed.has(id) || pending.has(id)) {
      return;
    }

    pending.add(id);

    try {
      await followArtist(id);
      followed.add(id);
    } catch {
      // keep the button enabled so the user can retry
    } finally {
      pending.delete(id);
    }
  }

  async function submit(event: SubmitEvent) {
    event.preventDefault();

    const q = term.trim();

    if (!q || searching) {
      return;
    }

    search = { status: "searching" };

    try {
      search = { status: "done", artists: await searchArtists(q) };
    } catch (err) {
      search = {
        status: "error",
        message: err instanceof Error ? err.message : String(err),
      };
    }
  }
</script>

<main>
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
  </header>

  <form class="search" onsubmit={submit}>
    <svg class="search-icon" viewBox="0 0 24 24" aria-hidden="true">
      <circle cx="11" cy="11" r="7" fill="none" stroke="currentColor" stroke-width="2" />
      <path d="M16.5 16.5 21 21" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
    </svg>
    <input bind:value={term} placeholder="Search an artist…" />
    <button disabled={searching}>
      {#if searching}<span class="spinner" aria-hidden="true"></span>{/if}
      {searching ? "Searching…" : "Search"}
    </button>
  </form>

  {#if search.status === "searching"}
    <div class="skeletons" aria-hidden="true">
      {#each { length: 4 } as _, i}
        <div class="skeleton" style="--i: {i}">
          <div class="skeleton-avatar"></div>
          <div class="skeleton-lines">
            <div class="skeleton-line"></div>
            <div class="skeleton-line short"></div>
          </div>
        </div>
      {/each}
    </div>
  {:else if search.status === "error"}
    <div class="notice" role="alert">
      <svg class="notice-icon error-icon" viewBox="0 0 24 24" aria-hidden="true">
        <circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" stroke-width="2" />
        <path d="M12 7.5v5.5" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
        <circle cx="12" cy="16.5" r="1.25" fill="currentColor" />
      </svg>
      <p class="error">{search.message}</p>
      <p class="hint">MusicBrainz may be rate-limiting; try again in a moment.</p>
    </div>
  {:else if search.status === "done" && search.artists.length > 0}
    <section class="results">
      <p class="count">
        {search.artists.length}
        {search.artists.length === 1 ? "artist" : "artists"}
      </p>
      <ul>
        {#each search.artists as artist, i (artist.id)}
          <li style="--i: {i}">
            <span class="avatar" aria-hidden="true">{artist.name.charAt(0).toUpperCase()}</span>
            <div class="who">
              <span class="name">{artist.name}</span>
              {#if artist.disambiguation}
                <span class="disambiguation">{artist.disambiguation}</span>
              {/if}
            </div>
            {#if followed.has(artist.id)}
              <span class="following">
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
            {:else}
              <button
                class="follow"
                onclick={() => follow(artist.id)}
                disabled={pending.has(artist.id)}
              >
                Follow
              </button>
            {/if}
          </li>
        {/each}
      </ul>
    </section>
  {:else if search.status === "done"}
    <div class="notice">
      <svg class="notice-icon" viewBox="0 0 24 24" aria-hidden="true">
        <circle cx="11" cy="11" r="7" fill="none" stroke="currentColor" stroke-width="2" />
        <path d="M16.5 16.5 21 21" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
        <path d="M8.5 11h5" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
      </svg>
      <p class="empty">No artists found.</p>
      <p class="hint">Try a different spelling or another name.</p>
    </div>
  {/if}
</main>
