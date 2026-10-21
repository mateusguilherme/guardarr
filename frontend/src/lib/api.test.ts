import { afterEach, expect, it, vi } from "vitest";
import { followArtist, followedArtists, searchArtists, type Artist } from "./api";

const caetano: Artist = { id: "abc", name: "Caetano Veloso" };

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

it("fetches the followed artist ids", async () => {
  const fetchMock = stubFetch(["abc", "def"]);

  const ids = await followedArtists();

  expect(fetchMock).toHaveBeenCalledWith("/api/followed");
  expect(ids).toEqual(["abc", "def"]);
});

it("throws when loading followed artists fails", async () => {
  stubFetch(null, false, 500);

  await expect(followedArtists()).rejects.toThrow(
    "loading followed artists failed (500)",
  );
});

it("follows an artist with a put request", async () => {
  const fetchMock = stubFetch(null);

  await followArtist("abc");

  expect(fetchMock).toHaveBeenCalledWith("/api/followed/abc", {
    method: "PUT",
  });
});

it("throws when following fails", async () => {
  stubFetch(null, false, 500);

  await expect(followArtist("abc")).rejects.toThrow("follow failed (500)");
});
