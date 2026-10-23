import { render, screen } from "@testing-library/svelte";
import { createRawSnippet } from "svelte";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import Layout from "./+layout.svelte";

// The header reads the current route off SvelteKit's page state, which only
// a running app fills in.
const page = vi.hoisted(() => ({ route: { id: "/" as string | null } }));

vi.mock("$app/state", () => ({ page }));
vi.mock("$app/navigation", () => ({ goto: vi.fn() }));

const children = createRawSnippet(() => ({ render: () => "<p>the page</p>" }));

function renderLayout() {
  render(Layout, { props: { children, params: {}, data: {} } });
}

beforeEach(() => {
  page.route.id = "/";
  // the app state loads the followed artists as the layout starts
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("[]")));
});

afterEach(() => {
  vi.unstubAllGlobals();
});

it("shows the page under the header", () => {
  renderLayout();

  expect(screen.getByText("the page")).toBeTruthy();
});

it("has the search in the header", () => {
  renderLayout();

  expect(screen.getByRole("search")).toBeTruthy();
});

it("leads home from the brand", () => {
  renderLayout();

  expect(screen.getByRole("link", { name: "guardarr" }).getAttribute("href")).toBe("/");
});

it("leads to the settings", () => {
  renderLayout();

  const link = screen.getByRole("link", { name: "Settings" });

  expect(link.getAttribute("href")).toBe("/settings");
  expect(link.getAttribute("aria-current")).toBeNull();
});

it("marks the settings as the current page while on them", () => {
  page.route.id = "/settings";
  renderLayout();

  expect(screen.getByRole("link", { name: "Settings" }).getAttribute("aria-current")).toBe("page");
});
