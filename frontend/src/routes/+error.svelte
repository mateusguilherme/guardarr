<script lang="ts">
  import { page } from "$app/state";
  import Lost from "#lib/Lost.svelte";
</script>

<Lost status={page.status} message={page.error?.message} />
