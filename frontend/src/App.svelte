<script lang="ts">
  type Artist = {
    id: string
    name: string
    disambiguation?: string
    score: number
  }

  let term = $state('')
  let artists = $state<Artist[]>([])
  let searching = $state(false)
  let error = $state('')
  let searched = $state(false)

  async function search(event: SubmitEvent) {
    event.preventDefault()

    const q = term.trim()
    if (!q || searching) return

    searching = true
    error = ''

    try {
      const response = await fetch(`/api/artists?q=${encodeURIComponent(q)}`)
      if (!response.ok) throw new Error(`search failed (${response.status})`)

      // The Go backend encodes an empty result as JSON null, not [].
      artists = (await response.json()) ?? []
      searched = true
    } catch (err) {
      error = err instanceof Error ? err.message : String(err)
      artists = []
    } finally {
      searching = false
    }
  }
</script>

<main>
  <h1>guardarr</h1>

  <form onsubmit={search}>
    <input bind:value={term} placeholder="Search an artist…" />
    <button disabled={searching}>{searching ? 'Searching…' : 'Search'}</button>
  </form>

  {#if error}
    <p class="error">{error}</p>
  {:else if artists.length > 0}
    <ul>
      {#each artists as artist (artist.id)}
        <li>
          <span class="name">{artist.name}</span>
          {#if artist.disambiguation}
            <span class="disambiguation">{artist.disambiguation}</span>
          {/if}
          <span class="score">{artist.score}</span>
        </li>
      {/each}
    </ul>
  {:else if searched && !searching}
    <p class="empty">No artists found.</p>
  {/if}
</main>
