<script lang="ts">
  import { resolve } from "$app/paths";
  import { saveSettings, slskdStatus, type SlskdStatus } from "#lib/api.ts";
  import type { PageProps } from "./$types";

  type Check = { state: "checking" } | { state: "failed" } | { state: "done"; status: SlskdStatus };

  let { data }: PageProps = $props();

  // The form shows the saved settings; what the user types overrides them
  // until the page loads them again.
  let url = $derived(data.slskd.url);
  let apiKey = $state("");
  let hasApiKey = $derived(data.slskd.hasApiKey);
  let saving = $state(false);
  let saved = $state(false);
  let saveError = $state<string>();
  let check = $state<Check>({ state: "checking" });
  // Bumped on every check, so a slow answer never overwrites a newer one.
  let checks = 0;

  let connection = $derived.by(() => {
    if (check.state === "checking") {
      return { tone: "checking", text: "Checking slskd…" };
    }

    if (check.state === "failed") {
      return { tone: "bad", text: "Could not check slskd." };
    }

    const { status } = check;

    if (!status.configured) {
      return { tone: "idle", text: "Not set up yet." };
    }

    if (!status.reachable) {
      return { tone: "bad", text: status.problem ?? "slskd did not answer." };
    }

    if (!status.loggedIn) {
      return { tone: "warn", text: "slskd answers, but it is not logged in to Soulseek." };
    }

    const as = status.username ? ` as ${status.username}` : "";

    return { tone: "ok", text: `Logged in to Soulseek${as}.` };
  });

  async function checkStatus() {
    const seq = ++checks;

    check = { state: "checking" };

    try {
      const status = await slskdStatus();

      if (seq === checks) {
        check = { state: "done", status };
      }
    } catch {
      if (seq === checks) {
        check = { state: "failed" };
      }
    }
  }

  checkStatus();

  async function save(event: SubmitEvent) {
    event.preventDefault();

    // A blank key field keeps the saved key.
    const key = apiKey.trim();

    saving = true;
    saved = false;
    saveError = undefined;

    try {
      await saveSettings({ slskd: { url: url.trim(), apiKey: key === "" ? undefined : key } });
      url = url.trim();
      apiKey = "";
      hasApiKey ||= key !== "";
      saved = true;
      checkStatus();
    } catch (err) {
      saveError = err instanceof Error ? err.message : String(err);
    } finally {
      saving = false;
    }
  }
</script>

<header class="page-header">
  <a class="back" href={resolve("/")} aria-label="Back to your artists">
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
  </a>
  <h2>Settings</h2>
</header>

<section class="panel" aria-labelledby="slskd-heading">
  <h3 id="slskd-heading">slskd</h3>
  <p class="panel-hint">
    Downloads go through slskd, the Soulseek client. guardarr reaches it at its web address, with
    one of its API keys.
  </p>

  <p class="connection {connection.tone}" role="status">
    <span class="dot" aria-hidden="true"></span>
    {connection.text}
  </p>

  <form class="settings-form" onsubmit={save} oninput={() => (saved = false)}>
    <div class="field">
      <label for="slskd-url">Address</label>
      <input
        id="slskd-url"
        aria-describedby="slskd-url-hint"
        type="url"
        bind:value={url}
        placeholder="http://localhost:5030"
        autocomplete="off"
        spellcheck="false"
      />
      <span class="field-hint" id="slskd-url-hint">
        Its web address, with any URL base it is set up with.
      </span>
    </div>

    <div class="field">
      <label for="slskd-api-key">API key</label>
      <input
        id="slskd-api-key"
        aria-describedby="slskd-api-key-hint"
        type="password"
        bind:value={apiKey}
        placeholder={hasApiKey ? "Saved; type a new one to replace it" : "One of the keys in slskd's config"}
        autocomplete="off"
      />
      <span class="field-hint" id="slskd-api-key-hint">
        Set in slskd under <code>web.authentication.api_keys</code>, or with
        <code>SLSKD_API_KEY</code>.
      </span>
    </div>

    {#if saveError}
      <p class="error" role="alert">{saveError}</p>
    {/if}

    <div class="form-actions">
      {#if saved}
        <span class="saved">Saved</span>
      {/if}
      <button type="submit" disabled={saving}>
        {#if saving}
          <span class="spinner" aria-hidden="true"></span>
        {/if}
        Save
      </button>
    </div>
  </form>
</section>
