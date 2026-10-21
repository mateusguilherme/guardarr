<script lang="ts">
  import { searchArtists, type Artist } from "./lib/api";

  type Search =
    | { status: "idle" }
    | { status: "searching" }
    | { status: "done"; artists: Artist[] }
    | { status: "error"; message: string };

  let term = $state("");
  let search = $state<Search>({ status: "idle" });
  let searching = $derived(search.status === "searching");

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
  <h1>guardarr</h1>

  <form onsubmit={submit}>
    <input bind:value={term} placeholder="Search an artist…" />
    <button disabled={searching}>{searching ? "Searching…" : "Search"}</button>
  </form>

  {#if search.status === "error"}
    <p class="error">{search.message}</p>
  {:else if search.status === "done" && search.artists.length > 0}
    <ul>
      {#each search.artists as artist (artist.id)}
        <li>
          <span class="name">{artist.name}</span>
          {#if artist.disambiguation}
            <span class="disambiguation">{artist.disambiguation}</span>
          {/if}
        </li>
      {/each}
    </ul>
  {:else if search.status === "done"}
    <p class="empty">No artists found.</p>
  {/if}
</main>
