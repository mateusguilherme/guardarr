import { afterEach, expect, it, vi } from "vitest";
import { searchArtists, type Artist } from "./api";

const caetano: Artist = { id: "abc", name: "Caetano Veloso", score: 100 };

function stubFetch(body: unknown, ok = true, status = 200) {
  const fetchMock = vi
    .fn()
    .mockResolvedValue({ ok, status, json: () => Promise.resolve(body) });

  vi.stubGlobal("fetch", fetchMock);

  return fetchMock;
}

afterEach(() => {
  vi.unstubAllGlobals();
});

it("fetches and decodes artists", async () => {
  const fetchMock = stubFetch([caetano]);

  const artists = await searchArtists("caetano veloso");

  expect(fetchMock).toHaveBeenCalledWith("/api/artists?q=caetano%20veloso");
  expect(artists).toEqual([caetano]);
});

it("throws on a non-ok response", async () => {
  stubFetch(null, false, 502);

  await expect(searchArtists("x")).rejects.toThrow("search failed (502)");
});
