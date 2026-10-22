import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "vitest";
import { AppState, appStateContext } from "#lib/app-state.svelte.ts";
import Page from "./+page.svelte";

type StubOptions = {
  artists?: unknown;
  artist?: unknown;
  followed?: unknown;
  releases?: unknown;
  ok?: boolean;
  status?: number;
};

function stubFetch({
  artists = [],
  artist = {},
  followed = [],
  releases = [],
  ok = true,
  status = 200,
}: StubOptions = {}) {
  const fetchMock = vi.fn((url: string) => {
    const body = url.endsWith("/releases")
      ? releases
      : url.startsWith("/api/followed")
        ? followed
        : url.startsWith("/api/artists?")
          ? artists
          : artist;

    return Promise.resolve({ ok, status, json: () => Promise.resolve(body) });
  });

  vi.stubGlobal("fetch", fetchMock);

  return fetchMock;
}

function renderHome(app = new AppState()) {
  return render(Page, { context: appStateContext(app) });
}

async function searchFor(term: string) {
  await userEvent.type(screen.getByPlaceholderText("Search an artist…"), term);
  await userEvent.click(screen.getByRole("button", { name: "Search" }));
}

afterEach(() => {
  vi.unstubAllGlobals();
});

it("shows nothing before the first search", () => {
  stubFetch();
  renderHome();

  expect(screen.queryByText("No artists found.")).toBeNull();
  expect(screen.queryByRole("list")).toBeNull();
});

it("lists the artists a search returns", async () => {
  stubFetch({
    artists: [
      { id: "abc", name: "Caetano Veloso" },
      { id: "def", name: "Caetaneando", disambiguation: "tribute trio" },
    ],
  });
  renderHome();

  await searchFor("caetano");

  expect(await screen.findByText("Caetano Veloso")).toBeDefined();
  expect(screen.getByText("Caetaneando")).toBeDefined();
  expect(screen.getByText("tribute trio")).toBeDefined();
});

it("shows a message when the search finds nothing", async () => {
  stubFetch();
  renderHome();

  await searchFor("zzzz");

  expect(await screen.findByText("No artists found.")).toBeDefined();
  // the icon's shapes arrive through a snippet and must still be SVG
  expect(document.querySelector(".notice-icon path")).toBeInstanceOf(SVGElement);
});

it("shows the error when the search fails", async () => {
  stubFetch({ ok: false, status: 503 });
  renderHome();

  await searchFor("caetano");

  expect(await screen.findByText("search failed (503)")).toBeDefined();
  expect(screen.getByRole("alert")).toBeDefined();
  expect(document.querySelector(".error-icon circle")).toBeInstanceOf(SVGElement);
  expect(screen.queryByText("No artists found.")).toBeNull();
});

it("does not search with an empty term", async () => {
  const fetchMock = stubFetch();

  renderHome();

  await userEvent.click(screen.getByRole("button", { name: "Search" }));

  const urls = fetchMock.mock.calls.map(([url]) => url);

  expect(urls.filter((url) => url.startsWith("/api/artists"))).toEqual([]);
});

it("marks artists that are already followed", async () => {
  stubFetch({
    artists: [
      { id: "abc", name: "Caetano Veloso" },
      { id: "def", name: "Djavan" },
    ],
    followed: [{ id: "abc", name: "Caetano Veloso" }],
  });
  renderHome();

  await searchFor("caetano");
  await screen.findByText("Djavan");

  expect(screen.getByText("Following")).toBeDefined();
  expect(screen.getByRole("button", { name: "Follow" })).toBeDefined();
});

it("follows an artist from the results", async () => {
  const fetchMock = stubFetch({ artists: [{ id: "abc", name: "Caetano Veloso" }] });

  renderHome();

  await searchFor("caetano");
  await screen.findByText("Caetano Veloso");
  await userEvent.click(screen.getByRole("button", { name: "Follow" }));

  expect(await screen.findByText("Following")).toBeDefined();
  expect(fetchMock).toHaveBeenCalledWith("/api/followed/abc", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: '{"name":"Caetano Veloso"}',
  });
  expect(screen.queryByRole("button", { name: "Follow" })).toBeNull();
});

it("unfollows an artist from the results", async () => {
  const fetchMock = stubFetch({
    artists: [{ id: "abc", name: "Caetano Veloso" }],
    followed: [{ id: "abc", name: "Caetano Veloso" }],
  });

  renderHome();

  await searchFor("caetano");
  await screen.findByText("Caetano Veloso");
  await userEvent.click(screen.getByText("Following"));

  expect(await screen.findByRole("button", { name: "Follow" })).toBeDefined();
  expect(fetchMock).toHaveBeenCalledWith("/api/followed/abc", { method: "DELETE" });
  expect(screen.queryByText("Following")).toBeNull();
});

it("unfollowing on the home removes the card", async () => {
  stubFetch({
    followed: [
      { id: "abc", name: "Caetano Veloso" },
      { id: "def", name: "Djavan" },
    ],
  });

  renderHome();

  await screen.findByText("Caetano Veloso");
  await userEvent.click(
    screen.getByRole("button", { name: "Unfollow Caetano Veloso" }),
  );

  expect(await screen.findByText("1 artist followed")).toBeDefined();
  expect(screen.queryByText("Caetano Veloso")).toBeNull();
  expect(screen.getByText("Djavan")).toBeDefined();
});

it("lists the followed artists on the home", async () => {
  stubFetch({
    followed: [
      { id: "abc", name: "Caetano Veloso" },
      { id: "def", name: "Caetaneando", disambiguation: "tribute trio" },
    ],
  });
  renderHome();

  expect(await screen.findByText("Caetano Veloso")).toBeDefined();
  expect(screen.getByText("tribute trio")).toBeDefined();
  expect(screen.getByText("2 artists followed")).toBeDefined();
});

it("escape clears the search and shows the followed artists again", async () => {
  stubFetch({
    artists: [{ id: "abc", name: "Caetano Veloso" }],
    followed: [{ id: "def", name: "Djavan" }],
  });
  renderHome();

  await screen.findByText("Djavan");
  await searchFor("caetano");
  await screen.findByText("Caetano Veloso");

  const input = screen.getByPlaceholderText<HTMLInputElement>("Search an artist…");

  await userEvent.type(input, "{Escape}");

  expect(input.value).toBe("");
  expect(screen.queryByText("Caetano Veloso")).toBeNull();
  expect(await screen.findByText("Djavan")).toBeDefined();
});

it("returns to the followed artists when the text is erased", async () => {
  stubFetch({
    artists: [{ id: "abc", name: "Caetano Veloso" }],
    followed: [{ id: "def", name: "Djavan" }],
  });
  renderHome();

  await screen.findByText("Djavan");
  await searchFor("caetano");
  await screen.findByText("Caetano Veloso");

  await userEvent.clear(screen.getByPlaceholderText("Search an artist…"));

  expect(await screen.findByText("Djavan")).toBeDefined();
  expect(screen.queryByText("Caetano Veloso")).toBeNull();
});

it("a second escape leaves the search field", async () => {
  stubFetch();
  renderHome();

  const input = screen.getByPlaceholderText<HTMLInputElement>("Search an artist…");

  await userEvent.type(input, "caetano");
  await userEvent.keyboard("{Escape}");

  expect(input.value).toBe("");
  expect(document.activeElement).toBe(input);

  await userEvent.keyboard("{Escape}");

  expect(document.activeElement).not.toBe(input);
});

it("filters the followed artists while typing, ignoring accents", async () => {
  const fetchMock = stubFetch({
    followed: [
      { id: "abc", name: "José Veloso" },
      { id: "def", name: "Djavan" },
    ],
  });
  renderHome();

  await screen.findByText("Djavan");
  await userEvent.type(
    screen.getByPlaceholderText("Search an artist…"),
    "jose",
  );

  expect(screen.getByText("José Veloso")).toBeDefined();
  expect(screen.queryByText("Djavan")).toBeNull();
  expect(screen.getByText("1 of 2 followed")).toBeDefined();

  const urls = fetchMock.mock.calls.map(([url]) => url);

  expect(urls.filter((url) => url.startsWith("/api/artists"))).toEqual([]);
});

it("offers the full search when no followed artist matches", async () => {
  stubFetch({ followed: [{ id: "abc", name: "Caetano Veloso" }] });
  renderHome();

  await screen.findByText("Caetano Veloso");
  await userEvent.type(
    screen.getByPlaceholderText("Search an artist…"),
    "zzzz",
  );

  expect(screen.queryByText("Caetano Veloso")).toBeNull();
  expect(
    screen.getByText("No followed artists match — press Enter to search MusicBrainz."),
  ).toBeDefined();
});

it("links each artist to its page", async () => {
  stubFetch({ followed: [{ id: "abc", name: "Caetano Veloso" }] });
  renderHome();

  const link = await screen.findByRole("link", { name: "Caetano Veloso" });

  expect(link.getAttribute("href")).toBe("/artist/abc");
});

it("keeps the search across a visit to an artist", async () => {
  stubFetch({ artists: [{ id: "abc", name: "Caetano Veloso" }] });

  const app = new AppState();
  const home = renderHome(app);

  await searchFor("caetano");
  await screen.findByText("Caetano Veloso");

  // Opening an artist unmounts the home; the layout keeps the state.
  home.unmount();
  renderHome(app);

  expect(screen.getByText("Caetano Veloso")).toBeDefined();
  expect(
    screen.getByPlaceholderText<HTMLInputElement>("Search an artist…").value,
  ).toBe("caetano");
});

it("hides the followed list once a search runs", async () => {
  stubFetch({ followed: [{ id: "abc", name: "Caetano Veloso" }] });
  renderHome();

  await screen.findByText("Caetano Veloso");
  await searchFor("zzzz");

  expect(await screen.findByText("No artists found.")).toBeDefined();
  expect(screen.queryByText("Caetano Veloso")).toBeNull();
});
