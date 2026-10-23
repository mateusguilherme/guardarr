export type Artist = {
  id: string;
  name: string;
  disambiguation?: string;
};

export function lookupArtist(id: string): Promise<Artist> {
  return request(`/api/artists/${encodeURIComponent(id)}`, "loading the artist failed");
}

export function searchArtists(q: string): Promise<Artist[]> {
  return request(`/api/artists?q=${encodeURIComponent(q)}`, "search failed");
}

export type Release = {
  id: string;
  title: string;
  firstReleaseDate?: string;
};

// A load function passes in the fetch SvelteKit gives it, which the dev
// server otherwise warns about.
export function artistReleases(id: string, fetch = globalThis.fetch): Promise<Release[]> {
  return request(
    `/api/artists/${encodeURIComponent(id)}/releases`,
    "loading releases failed",
    { fetch },
  );
}

export function followedArtists(): Promise<Artist[]> {
  return request("/api/followed", "loading followed artists failed");
}

export function followArtist(artist: Artist): Promise<void> {
  return request(`/api/followed/${encodeURIComponent(artist.id)}`, "follow failed", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      name: artist.name,
      disambiguation: artist.disambiguation,
    }),
  });
}

export function unfollowArtist(id: string): Promise<void> {
  return request(`/api/followed/${encodeURIComponent(id)}`, "unfollow failed", {
    method: "DELETE",
  });
}

// Settings come back whole, one section per thing to set up.
export type Settings = {
  slskd: SlskdSettings;
};

// The saved API key never comes back; hasApiKey says whether there is one.
export type SlskdSettings = {
  url: string;
  hasApiKey: boolean;
};

export function settings(fetch = globalThis.fetch): Promise<Settings> {
  return request("/api/settings", "loading the settings failed", { fetch });
}

// SettingsChanges are the sections to save; the ones left out stay as they
// are, and so does the saved slskd key when apiKey is left out.
export type SettingsChanges = {
  slskd?: { url: string; apiKey?: string };
};

export function saveSettings(changes: SettingsChanges): Promise<void> {
  return request("/api/settings", "saving the settings failed", {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(changes),
  });
}

// problem says, in words to show, why a configured slskd is not reachable.
export type SlskdStatus = {
  configured: boolean;
  reachable: boolean;
  loggedIn: boolean;
  username?: string;
  problem?: string;
};

export function slskdStatus(): Promise<SlskdStatus> {
  return request("/api/slskd/status", "checking slskd failed");
}

// Options are fetch's own, plus the fetch to call them with.
type Options = RequestInit & { fetch?: typeof globalThis.fetch };

// request fetches an API path and decodes its body, unless the response is
// not ok: then it throws an error that says what failed and with which status.
async function request<T>(
  path: string,
  failure: string,
  { fetch = globalThis.fetch, ...init }: Options = {},
): Promise<T> {
  const response = await fetch(path, init);

  if (!response.ok) {
    throw new Error(`${failure} (${response.status})`);
  }

  // following and unfollowing answer 204, with no body to decode
  return response.status === 204 ? (undefined as T) : response.json();
}
