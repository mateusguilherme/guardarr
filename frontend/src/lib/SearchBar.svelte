<script lang="ts">
  import { goto } from "$app/navigation";
  import { resolve } from "$app/paths";
  import { page } from "$app/state";
  import { getAppState } from "#lib/app-state.svelte.ts";

  const app = getAppState();

  let searching = $derived(app.search.status === "searching");

  // The results show on the home, so a search started on any other page
  // goes there first.
  async function submit(event: SubmitEvent) {
    event.preventDefault();

    if (app.term.trim() === "") {
      return;
    }

    if (page.route.id !== "/") {
      await goto(resolve("/"));
    }

    app.runSearch();
  }

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

<form class="search" role="search" onsubmit={submit}>
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
