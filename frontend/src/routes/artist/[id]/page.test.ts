import { render, screen } from "@testing-library/svelte";
import { afterEach, expect, it, vi } from "vitest";
import type { Release } from "#lib/api.ts";
import { AppState, appStateContext } from "#lib/app-state.svelte.ts";
import Page from "./+page.svelte";
import { load } from "./+page";

type StubOptions = {
  artist?: unknown;
  followed?: unknown;
  followedOk?: boolean;
};

// Only the page's own lookups go through this fetch; the releases arrive
// through data, as the load function hands them over.
function stubFetch({
  artist = {},
  followed = [],
  followedOk = true,
}: StubOptions = {}) {
  const fetchMock = vi.fn((url: string) => {
    if (url === "/api/followed") {
      return Promise.resolve({
        ok: followedOk,
        status: followedOk ? 200 : 500,
        json: () => Promise.resolve(followed),
      });
    }

    return Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve(artist) });
  });

  vi.stubGlobal("fetch", fetchMock);

  return fetchMock;
}

function renderArtist({
  releases = Promise.resolve<Release[]>([]),
  app = new AppState(),
} = {}) {
  render(Page, {
    props: { params: { id: "abc" }, data: { releases } },
    context: appStateContext(app),
  });
}

function lookups(fetchMock: ReturnType<typeof stubFetch>) {
  return fetchMock.mock.calls
    .map(([url]) => url)
    .filter((url) => url === "/api/artists/abc");
}

afterEach(() => {
  vi.unstubAllGlobals();
});

it("loads the releases with the fetch SvelteKit passes in", async () => {
  const fetchMock = vi
    .fn()
    .mockResolvedValue({ ok: true, status: 200, json: () => Promise.resolve([]) });

  const data = await load({
    params: { id: "abc" },
    fetch: fetchMock,
  } as unknown as Parameters<typeof load>[0]);

  expect(await data?.releases).toEqual([]);
  expect(fetchMock).toHaveBeenCalledWith("/api/artists/abc/releases", {});
});

it("lists the artist's releases", async () => {
  stubFetch({ followed: [{ id: "abc", name: "Caetano Veloso" }] });
  renderArtist({
    releases: Promise.resolve([
      { id: "rg1", title: "Transa", firstReleaseDate: "1972-01-01" },
      { id: "rg2", title: "Araçá Azul" },
    ]),
  });

  expect(await screen.findByText("Transa")).toBeDefined();
  expect(screen.getByText("1972")).toBeDefined();
  expect(screen.getByText("Araçá Azul")).toBeDefined();
  expect(screen.getByText("2 official releases")).toBeDefined();
});

it("shows a message when the artist has no releases", async () => {
  stubFetch({ followed: [{ id: "abc", name: "Caetano Veloso" }] });
  renderArtist();

  expect(await screen.findByText("No official releases.")).toBeDefined();
});

it("shows the error when loading releases fails", async () => {
  stubFetch({ followed: [{ id: "abc", name: "Caetano Veloso" }] });
  renderArtist({
    releases: Promise.reject(new Error("loading releases failed (503)")),
  });

  expect(await screen.findByText("loading releases failed (503)")).toBeDefined();
});

it("links back home", () => {
  stubFetch();
  renderArtist();

  const back = screen.getByRole("link", { name: "Back to your artists" });

  expect(back.getAttribute("href")).toBe("/");
});

it("uses the name of a followed artist", async () => {
  const fetchMock = stubFetch({
    followed: [{ id: "abc", name: "Caetano Veloso" }],
  });

  renderArtist();

  expect(await screen.findByRole("heading", { name: "Caetano Veloso" })).toBeDefined();
  expect(lookups(fetchMock)).toEqual([]);
});

it("uses the name from the latest search", async () => {
  const fetchMock = stubFetch();
  const app = new AppState();

  app.search = { status: "done", artists: [{ id: "abc", name: "Caetano Veloso" }] };
  renderArtist({ app });

  expect(screen.getByRole("heading", { name: "Caetano Veloso" })).toBeDefined();

  // the lookup would only start once the followed list settles
  await vi.waitFor(() => expect(app.followedLoaded).toBe(true));

  expect(lookups(fetchMock)).toEqual([]);
});

it("a deep link looks the artist's name up", async () => {
  const fetchMock = stubFetch({ artist: { id: "abc", name: "Caetano Veloso" } });

  renderArtist();

  expect(await screen.findByRole("heading", { name: "Caetano Veloso" })).toBeDefined();
  expect(lookups(fetchMock)).toEqual(["/api/artists/abc"]);
});

it("a deep link loads the followed artists only once", async () => {
  const fetchMock = stubFetch({ artist: { id: "abc", name: "Caetano Veloso" } });

  renderArtist();

  await screen.findByRole("heading", { name: "Caetano Veloso" });

  const urls = fetchMock.mock.calls.map(([url]) => url);

  expect(urls.filter((url) => url === "/api/followed")).toEqual(["/api/followed"]);
});

it("a deep link still finds the name when the followed list fails", async () => {
  stubFetch({
    artist: { id: "abc", name: "Caetano Veloso" },
    followedOk: false,
  });
  renderArtist();

  expect(await screen.findByRole("heading", { name: "Caetano Veloso" })).toBeDefined();
});
