<script lang="ts">
  import "../app.css";
  import { resolve } from "$app/paths";
  import { AppState, setAppState } from "#lib/app-state.svelte.ts";
  import SearchBar from "#lib/SearchBar.svelte";
  import type { LayoutProps } from "./$types";

  let { children }: LayoutProps = $props();

  setAppState(new AppState());
</script>

<header class="app-header">
  <div class="app-header-content">
    <a class="brand" href={resolve("/")}>
      <svg class="mark" viewBox="0 0 24 24" aria-hidden="true">
        <circle cx="10" cy="10" r="7.25" fill="none" stroke="currentColor" stroke-width="2.5" />
        <circle cx="10" cy="10" r="4" fill="none" stroke="currentColor" stroke-width="1" opacity="0.5" />
        <circle cx="10" cy="10" r="1.5" fill="currentColor" />
        <path d="M15.5 15.5 21 21" stroke="currentColor" stroke-width="2.75" stroke-linecap="round" />
      </svg>
      <span>guard<span class="arr">arr</span></span>
    </a>
    <SearchBar />
  </div>
</header>

<main>
  {@render children()}
</main>
