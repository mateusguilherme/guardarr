<script lang="ts">
  import { resolve } from "$app/paths";
  import type { Artist } from "#lib/api.ts";
  import { getAppState } from "#lib/app-state.svelte.ts";
  import Notice from "#lib/Notice.svelte";

  const app = getAppState();

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

</script>

{#snippet artistList(artists: Artist[])}
  <ul class="artists">
    {#each artists as artist, i (artist.id)}
      <li style="--i: {i}">
        <a class="open-artist" href={resolve("/artist/[id=mbid]", { id: artist.id })}>
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
