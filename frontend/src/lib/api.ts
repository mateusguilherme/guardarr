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

export async function followedArtists(): Promise<Artist[]> {
  const response = await fetch("/api/followed");

  if (!response.ok) {
    throw new Error(`loading followed artists failed (${response.status})`);
  }

  return response.json();
}

export async function followArtist(artist: Artist): Promise<void> {
  const response = await fetch(`/api/followed/${encodeURIComponent(artist.id)}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      name: artist.name,
      disambiguation: artist.disambiguation,
    }),
  });

  if (!response.ok) {
    throw new Error(`follow failed (${response.status})`);
  }
}

export async function unfollowArtist(id: string): Promise<void> {
  const response = await fetch(`/api/followed/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });

  if (!response.ok) {
    throw new Error(`unfollow failed (${response.status})`);
  }
}
