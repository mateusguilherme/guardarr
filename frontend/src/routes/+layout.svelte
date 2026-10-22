<script lang="ts">
  import "../app.css";
  import { AppState, setAppState } from "#lib/app-state.svelte.ts";
  import type { LayoutProps } from "./$types";

  let { children }: LayoutProps = $props();

  setAppState(new AppState());
</script>

<main>
  {@render children()}
</main>
