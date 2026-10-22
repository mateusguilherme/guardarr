<script lang="ts">
  import type { Snippet } from "svelte";

  // Every error here comes from a call that reaches MusicBrainz, so an
  // error notice brings its own icon and hint; an empty state draws its
  // icon and says what to try instead.
  type Props =
    | { error: true; message: string; hint?: never; icon?: never }
    | { error?: false; message: string; hint: string; icon: Snippet };

  let {
    error = false,
    message,
    hint = "MusicBrainz may be rate-limiting; try again in a moment.",
    icon = errorIcon,
  }: Props = $props();
</script>

{#snippet errorIcon()}
  <circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" stroke-width="2" />
  <path d="M12 7.5v5.5" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
  <circle cx="12" cy="16.5" r="1.25" fill="currentColor" />
{/snippet}

<div class="notice" role={error ? "alert" : undefined}>
  <svg class="notice-icon" class:error-icon={error} viewBox="0 0 24 24" aria-hidden="true">
    {@render icon()}
  </svg>
  <p class={error ? "error" : "empty"}>{message}</p>
  <p class="hint">{hint}</p>
</div>
