export type Artist = {
  id: string;
  name: string;
  disambiguation?: string;
};

export async function searchArtists(q: string): Promise<Artist[]> {
  const response = await fetch(`/api/artists?q=${encodeURIComponent(q)}`);

  if (!response.ok) {
    throw new Error(`search failed (${response.status})`);
  }

  return response.json();
}

export async function followedArtists(): Promise<string[]> {
  const response = await fetch("/api/followed");

  if (!response.ok) {
    throw new Error(`loading followed artists failed (${response.status})`);
  }

  return response.json();
}

export async function followArtist(id: string): Promise<void> {
  const response = await fetch(`/api/followed/${encodeURIComponent(id)}`, {
    method: "PUT",
  });

  if (!response.ok) {
    throw new Error(`follow failed (${response.status})`);
  }
}
