import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "vitest";
import App from "./App.svelte";

type StubOptions = {
  artists?: unknown;
  followed?: unknown;
  ok?: boolean;
  status?: number;
};

function stubFetch({
  artists = [],
  followed = [],
  ok = true,
  status = 200,
}: StubOptions = {}) {
  const fetchMock = vi.fn((url: string) => {
    const body = url.startsWith("/api/followed") ? followed : artists;

    return Promise.resolve({ ok, status, json: () => Promise.resolve(body) });
  });

  vi.stubGlobal("fetch", fetchMock);

  return fetchMock;
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
  render(App);

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
  render(App);

  await searchFor("caetano");

  expect(await screen.findByText("Caetano Veloso")).toBeDefined();
  expect(screen.getByText("Caetaneando")).toBeDefined();
  expect(screen.getByText("tribute trio")).toBeDefined();
});

it("shows a message when the search finds nothing", async () => {
  stubFetch();
  render(App);

  await searchFor("zzzz");

  expect(await screen.findByText("No artists found.")).toBeDefined();
});

it("shows the error when the search fails", async () => {
  stubFetch({ ok: false, status: 503 });
  render(App);

  await searchFor("caetano");

  expect(await screen.findByText("search failed (503)")).toBeDefined();
  expect(screen.queryByText("No artists found.")).toBeNull();
});

it("does not search with an empty term", async () => {
  const fetchMock = stubFetch();

  render(App);

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
    followed: ["abc"],
  });
  render(App);

  await searchFor("caetano");
  await screen.findByText("Caetano Veloso");

  expect(screen.getByText("Following")).toBeDefined();
  expect(screen.getByRole("button", { name: "Follow" })).toBeDefined();
});

it("follows an artist from the results", async () => {
  const fetchMock = stubFetch({ artists: [{ id: "abc", name: "Caetano Veloso" }] });

  render(App);

  await searchFor("caetano");
  await screen.findByText("Caetano Veloso");
  await userEvent.click(screen.getByRole("button", { name: "Follow" }));

  expect(await screen.findByText("Following")).toBeDefined();
  expect(fetchMock).toHaveBeenCalledWith("/api/followed/abc", { method: "PUT" });
  expect(screen.queryByRole("button", { name: "Follow" })).toBeNull();
});
