import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { AppState, appStateContext } from "#lib/app-state.svelte.ts";
import SearchBar from "./SearchBar.svelte";

// The bar asks SvelteKit which page it is on, and goes home through it.
const page = vi.hoisted(() => ({ route: { id: "/" as string | null } }));
const goto = vi.hoisted(() => vi.fn(() => Promise.resolve()));

vi.mock("$app/state", () => ({ page }));
vi.mock("$app/navigation", () => ({ goto }));

function stubFetch() {
  // the app state loads the followed artists, and a search finds no one
  const fetchMock = vi.fn<(url: string) => Promise<Response>>(() =>
    Promise.resolve(new Response("[]")),
  );

  vi.stubGlobal("fetch", fetchMock);

  return fetchMock;
}

function searches(fetchMock: ReturnType<typeof stubFetch>) {
  return fetchMock.mock.calls.map(([url]) => url).filter((url) => url.startsWith("/api/artists?"));
}

function renderBar() {
  render(SearchBar, { context: appStateContext(new AppState()) });
}

beforeEach(() => {
  page.route.id = "/";
  goto.mockClear();
});

afterEach(() => {
  vi.unstubAllGlobals();
});

it("searches in place on the home", async () => {
  const fetchMock = stubFetch();

  renderBar();

  await userEvent.type(screen.getByPlaceholderText("Search an artist…"), "caetano{Enter}");

  expect(goto).not.toHaveBeenCalled();
  expect(searches(fetchMock)).toEqual(["/api/artists?q=caetano"]);
});

it("goes home to show a search started on another page", async () => {
  const fetchMock = stubFetch();

  page.route.id = "/artist/[id]";
  renderBar();

  await userEvent.type(screen.getByPlaceholderText("Search an artist…"), "caetano{Enter}");

  expect(goto).toHaveBeenCalledWith("/");
  expect(searches(fetchMock)).toEqual(["/api/artists?q=caetano"]);
});

it("stays on the page for an empty search", async () => {
  const fetchMock = stubFetch();

  page.route.id = "/settings";
  renderBar();

  await userEvent.click(screen.getByRole("button", { name: "Search" }));

  expect(goto).not.toHaveBeenCalled();
  expect(searches(fetchMock)).toEqual([]);
});
