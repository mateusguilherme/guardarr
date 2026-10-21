<script lang="ts">
  import { SvelteMap, SvelteSet } from "svelte/reactivity";
  import {
    followArtist,
    followedArtists,
    searchArtists,
    unfollowArtist,
    type Artist,
  } from "./lib/api";
  import ArtistPage from "./lib/ArtistPage.svelte";

  type Route = { page: "home" } | { page: "artist"; id: string };

  // The hash is the router: "#/artist/<id>" opens an artist's releases and
  // anything else is the home, so links survive reloads and the browser's
  // back button keeps working.
  function parseRoute(hash: string): Route {
    const match = hash.match(/^#\/artist\/([^/]+)/);

    if (match) {
      return { page: "artist", id: decodeURIComponent(match[1]) };
    }

    return { page: "home" };
  }

  let route = $state(parseRoute(location.hash));

  type Search =
    | { status: "idle" }
    | { status: "searching" }
    | { status: "done"; artists: Artist[] }
    | { status: "error"; message: string };

  let term = $state("");
  let search = $state<Search>({ status: "idle" });
  let searching = $derived(search.status === "searching");

  const followed = new SvelteMap<string, Artist>();
  const pending = new SvelteSet<string>();

  // The artist page usually gets its name from state the home already
  // holds — the followed map or the latest search results — so opening it
  // costs no extra lookup.
  let routeArtist = $derived.by(() => {
    if (route.page !== "artist") {
      return undefined;
    }

    const { id } = route;

    return (
      followed.get(id) ??
      (search.status === "done"
        ? search.artists.find((artist) => artist.id === id)
        : undefined)
    );
  });

  // Failing to load the list only means an empty home and every button
  // starting as "Follow".
  followedArtists().then(
    (artists) => artists.forEach((artist) => followed.set(artist.id, artist)),
    () => {},
  );

  async function follow(artist: Artist) {
    if (followed.has(artist.id) || pending.has(artist.id)) {
      return;
    }

    pending.add(artist.id);

    try {
      await followArtist(artist);
      followed.set(artist.id, artist);
    } catch {
      // keep the button enabled so the user can retry
    } finally {
      pending.delete(artist.id);
    }
  }

  async function unfollow(id: string) {
    if (!followed.has(id) || pending.has(id)) {
      return;
    }

    pending.add(id);

    try {
      await unfollowArtist(id);
      followed.delete(id);
    } catch {
      // still following; the button stays enabled so the user can retry
    } finally {
      pending.delete(id);
    }
  }

  // Bumped on reset so the result of an in-flight search is discarded
  // instead of reappearing over the home.
  let searchSeq = 0;

  async function submit(event: SubmitEvent) {
    event.preventDefault();

    const q = term.trim();

    if (!q || searching) {
      return;
    }

    const seq = ++searchSeq;

    search = { status: "searching" };

    try {
      const artists = await searchArtists(q);

      if (seq === searchSeq) {
        search = { status: "done", artists };
      }
    } catch (err) {
      if (seq === searchSeq) {
        search = {
          status: "error",
          message: err instanceof Error ? err.message : String(err),
        };
      }
    }
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
    const q = fold(term.trim());
    const all = [...followed.values()];

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
    if (term.trim() === "" && search.status !== "idle") {
      searchSeq++;
      search = { status: "idle" };
    }
  });

  function keydown(event: KeyboardEvent) {
    if (event.key !== "Escape") {
      return;
    }

    // First escape clears the text (the effect above restores the home);
    // with nothing left to clear, a second one leaves the field.
    if (term === "") {
      (event.currentTarget as HTMLInputElement).blur();
      return;
    }

    term = "";
  }
</script>

{#snippet artistCard(artist: Artist, i: number)}
  <li style="--i: {i}">
    <a class="open-artist" href="#/artist/{artist.id}">
      <span class="avatar" aria-hidden="true">{artist.name.charAt(0).toUpperCase()}</span>
      <div class="who">
        <span class="name">{artist.name}</span>
        {#if artist.disambiguation}
          <span class="disambiguation">{artist.disambiguation}</span>
        {/if}
      </div>
    </a>
    {#if followed.has(artist.id)}
      <button
        class="following"
        onclick={() => unfollow(artist.id)}
        disabled={pending.has(artist.id)}
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
        onclick={() => follow(artist)}
        disabled={pending.has(artist.id)}
      >
        Follow
      </button>
    {/if}
  </li>
{/snippet}

<svelte:window onhashchange={() => (route = parseRoute(location.hash))} />

<main>
  {#if route.page === "artist"}
    <ArtistPage id={route.id} name={routeArtist?.name} />
  {:else}
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
      <input bind:value={term} placeholder="Search an artist…" onkeydown={keydown} />
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
            {@render artistCard(artist, i)}
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
    {:else if search.status === "idle" && followed.size > 0}
      <section class="results">
        <p class="count">
          {#if term.trim()}
            {filteredFollowed.length} of {followed.size} followed
          {:else}
            {followed.size}
            {followed.size === 1 ? "artist followed" : "artists followed"}
          {/if}
        </p>
        {#if filteredFollowed.length > 0}
          <ul>
            {#each filteredFollowed as artist, i (artist.id)}
              {@render artistCard(artist, i)}
            {/each}
          </ul>
        {:else}
          <p class="no-match">
            No followed artists match — press Enter to search MusicBrainz.
          </p>
        {/if}
      </section>
    {/if}
  {/if}
</main>
