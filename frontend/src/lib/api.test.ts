import { afterEach, expect, it, vi } from "vitest";
import {
  artistReleases,
  followArtist,
  lookupArtist,
  followedArtists,
  searchArtists,
  unfollowArtist,
  type Artist,
} from "./api";

const caetano: Artist = { id: "abc", name: "Caetano Veloso" };

function stubFetch(body: unknown, ok = true, status = 200) {
  const fetchMock = vi
    .fn()
    .mockResolvedValue({ ok, status, json: () => Promise.resolve(body) });

  vi.stubGlobal("fetch", fetchMock);

  return fetchMock;
}

// Following and unfollowing answer 204: a real Response, whose empty body
// would fail to decode.
function stubNoContent() {
  const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));

  vi.stubGlobal("fetch", fetchMock);

  return fetchMock;
}

afterEach(() => {
  vi.unstubAllGlobals();
});

it("fetches and decodes artists", async () => {
  const fetchMock = stubFetch([caetano]);

  const artists = await searchArtists("caetano veloso");

  expect(fetchMock).toHaveBeenCalledWith("/api/artists?q=caetano%20veloso", {});
  expect(artists).toEqual([caetano]);
});

it("throws on a non-ok response", async () => {
  stubFetch(null, false, 502);

  await expect(searchArtists("x")).rejects.toThrow("search failed (502)");
});

it("looks up one artist", async () => {
  const fetchMock = stubFetch(caetano);

  const artist = await lookupArtist("abc");

  expect(fetchMock).toHaveBeenCalledWith("/api/artists/abc", {});
  expect(artist).toEqual(caetano);
});

it("throws when the artist lookup fails", async () => {
  stubFetch(null, false, 502);

  await expect(lookupArtist("abc")).rejects.toThrow(
    "loading the artist failed (502)",
  );
});

it("fetches an artist's releases", async () => {
  const transa = { id: "rg1", title: "Transa", firstReleaseDate: "1972-01-01" };
  const fetchMock = stubFetch([transa]);

  const releases = await artistReleases("abc");

  expect(fetchMock).toHaveBeenCalledWith("/api/artists/abc/releases", {});
  expect(releases).toEqual([transa]);
});

it("throws when loading releases fails", async () => {
  stubFetch(null, false, 502);

  await expect(artistReleases("abc")).rejects.toThrow("loading releases failed (502)");
});

it("fetches the followed artists", async () => {
  const fetchMock = stubFetch([caetano]);

  const artists = await followedArtists();

  expect(fetchMock).toHaveBeenCalledWith("/api/followed", {});
  expect(artists).toEqual([caetano]);
});

it("throws when loading followed artists fails", async () => {
  stubFetch(null, false, 500);

  await expect(followedArtists()).rejects.toThrow(
    "loading followed artists failed (500)",
  );
});

it("follows an artist with a put request", async () => {
  const fetchMock = stubNoContent();

  await expect(
    followArtist({ ...caetano, disambiguation: "Brazilian singer" }),
  ).resolves.toBeUndefined();

  expect(fetchMock).toHaveBeenCalledWith("/api/followed/abc", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: '{"name":"Caetano Veloso","disambiguation":"Brazilian singer"}',
  });
});

it("throws when following fails", async () => {
  stubFetch(null, false, 500);

  await expect(followArtist(caetano)).rejects.toThrow("follow failed (500)");
});

it("unfollows an artist with a delete request", async () => {
  const fetchMock = stubNoContent();

  await expect(unfollowArtist("abc")).resolves.toBeUndefined();

  expect(fetchMock).toHaveBeenCalledWith("/api/followed/abc", {
    method: "DELETE",
  });
});

it("throws when unfollowing fails", async () => {
  stubFetch(null, false, 500);

  await expect(unfollowArtist("abc")).rejects.toThrow("unfollow failed (500)");
});
