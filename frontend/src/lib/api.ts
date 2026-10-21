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
