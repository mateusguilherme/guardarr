<script lang="ts">
  import { resolve } from "$app/paths";

  // Lost fills the page when there is nothing to show at an address: the
  // error page renders it, and so does an artist MusicBrainz doesn't know.
  let { status, message }: { status: number; message?: string } = $props();

  let notFound = $derived(status === 404);
</script>

<section class="lost">
  <p class="lost-code" aria-hidden="true">
    {#if notFound}
      <span>4</span>
      <!-- the 0 is a record, spinning like the ones the app keeps -->
      <svg class="lost-record" viewBox="0 0 24 24">
        <circle cx="12" cy="12" r="10.25" fill="none" stroke="currentColor" stroke-width="2.5" />
        <path
          d="M5.29 8.13A7.75 7.75 0 0 1 9.35 4.72M18.71 15.88A7.75 7.75 0 0 1 14.65 19.28"
          fill="none"
          stroke="currentColor"
          stroke-width="1.25"
          stroke-linecap="round"
          opacity="0.55"
        />
        <circle cx="12" cy="12" r="5.5" fill="none" stroke="currentColor" stroke-width="0.75" opacity="0.4" />
        <circle cx="12" cy="12" r="3.25" fill="currentColor" />
        <circle class="lost-record-hole" cx="12" cy="12" r="0.9" />
      </svg>
      <span>4</span>
    {:else}
      {status}
    {/if}
  </p>

  {#if notFound}
    <h2>Nothing kept here.</h2>
    <p class="hint">This page doesn't exist; the link may be old or mistyped.</p>
  {:else}
    <h2>Something went wrong.</h2>
    <p class="hint">{message}</p>
  {/if}

  <a class="home-link" href={resolve("/")}>
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path
        d="M14.5 6 8.5 12l6 6"
        fill="none"
        stroke="currentColor"
        stroke-width="2.5"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>
    Back to your artists
  </a>
</section>
