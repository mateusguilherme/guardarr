<script lang="ts">
  import "../app.css";
  import { resolve } from "$app/paths";
  import { page } from "$app/state";
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
    <nav aria-label="Main">
      <a
        class="settings-link"
        href={resolve("/settings")}
        aria-label="Settings"
        aria-current={page.route.id === "/settings" ? "page" : undefined}
      >
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M4 7h9M19 7h1M4 17h3M13 17h7" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
          <circle cx="16" cy="7" r="2.25" fill="none" stroke="currentColor" stroke-width="2" />
          <circle cx="10" cy="17" r="2.25" fill="none" stroke="currentColor" stroke-width="2" />
        </svg>
      </a>
    </nav>
  </div>
</header>

<main>
  {@render children()}
</main>
