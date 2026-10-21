import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "vitest";
import App from "./App.svelte";

function stubFetch(body: unknown, ok = true, status = 200) {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue({ ok, status, json: () => Promise.resolve(body) }),
  );
}

async function searchFor(term: string) {
  await userEvent.type(screen.getByPlaceholderText("Search an artist…"), term);
  await userEvent.click(screen.getByRole("button"));
}

afterEach(() => {
  vi.unstubAllGlobals();
});

it("shows nothing before the first search", () => {
  stubFetch([]);
  render(App);

  expect(screen.queryByText("No artists found.")).toBeNull();
  expect(screen.queryByRole("list")).toBeNull();
});

it("lists the artists a search returns", async () => {
  stubFetch([
    { id: "abc", name: "Caetano Veloso", score: 100 },
    { id: "def", name: "Caetaneando", disambiguation: "tribute trio", score: 41 },
  ]);
  render(App);

  await searchFor("caetano");

  expect(await screen.findByText("Caetano Veloso")).toBeDefined();
  expect(screen.getByText("Caetaneando")).toBeDefined();
  expect(screen.getByText("tribute trio")).toBeDefined();
});

it("shows a message when the search finds nothing", async () => {
  stubFetch([]);
  render(App);

  await searchFor("zzzz");

  expect(await screen.findByText("No artists found.")).toBeDefined();
});

it("shows the error when the search fails", async () => {
  stubFetch(null, false, 503);
  render(App);

  await searchFor("caetano");

  expect(await screen.findByText("search failed (503)")).toBeDefined();
  expect(screen.queryByText("No artists found.")).toBeNull();
});

it("does not search with an empty term", async () => {
  const fetchMock = vi.fn();

  vi.stubGlobal("fetch", fetchMock);
  render(App);

  await userEvent.click(screen.getByRole("button"));

  expect(fetchMock).not.toHaveBeenCalled();
});
