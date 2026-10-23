import { render, screen } from "@testing-library/svelte";
import { beforeEach, expect, it, vi } from "vitest";
import ErrorPage from "./+error.svelte";

// The error page reads the status and error off SvelteKit's page state,
// which only a running app fills in.
const page = vi.hoisted(() => ({ status: 404, error: { message: "Not Found" } }));

vi.mock("$app/state", () => ({ page }));

beforeEach(() => {
  page.status = 404;
  page.error = { message: "Not Found" };
});

it("says nothing is kept at an unknown address", () => {
  render(ErrorPage);

  expect(screen.getByRole("heading", { name: "Nothing kept here." })).toBeTruthy();
  expect(screen.getByText(/This page doesn't exist/)).toBeTruthy();
});

it("leads back to the artists", () => {
  render(ErrorPage);

  const link = screen.getByRole("link", { name: "Back to your artists" });

  expect(link.getAttribute("href")).toBe("/");
});

it("shows any other error with its message", () => {
  page.status = 500;
  page.error = { message: "Internal Error" };

  render(ErrorPage);

  expect(screen.getByRole("heading", { name: "Something went wrong." })).toBeTruthy();
  expect(screen.getByText("Internal Error")).toBeTruthy();
  expect(screen.getByText("500")).toBeTruthy();
});
