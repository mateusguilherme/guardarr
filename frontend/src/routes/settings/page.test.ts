import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "vitest";
import type { Settings, SlskdSettings, SlskdStatus } from "#lib/api.ts";
import { load } from "./+page.ts";
import Page from "./+page.svelte";

const notSetUp: SlskdStatus = { configured: false, reachable: false, loggedIn: false };
const loggedIn: SlskdStatus = { configured: true, reachable: true, loggedIn: true, username: "guardarr" };

// stubFetch answers the page's calls: each status check with the next of
// statuses, the last one repeating, and a save with saveStatus.
function stubFetch({ statuses = [notSetUp], saveStatus = 204 } = {}) {
  let checks = 0;

  const fetchMock = vi.fn((_url: string, init?: RequestInit) => {
    if (init?.method === "PATCH") {
      return Promise.resolve(new Response(null, { status: saveStatus }));
    }

    const status = statuses[Math.min(checks++, statuses.length - 1)];

    return Promise.resolve(new Response(JSON.stringify(status)));
  });

  vi.stubGlobal("fetch", fetchMock);

  return fetchMock;
}

function renderSettings(slskd: SlskdSettings = { url: "", hasApiKey: false }) {
  render(Page, { props: { params: {}, data: { slskd } } });
}

// saves returns the bodies of the save requests, decoded.
function saves(fetchMock: ReturnType<typeof stubFetch>) {
  return fetchMock.mock.calls
    .filter(([, init]) => init?.method === "PATCH")
    .map(([, init]) => JSON.parse(String(init?.body)));
}

afterEach(() => {
  vi.unstubAllGlobals();
});

it("loads the saved settings with the fetch SvelteKit passes in", async () => {
  const settings: Settings = { slskd: { url: "http://localhost:5030", hasApiKey: true } };
  const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify(settings)));

  const data = await load({ fetch: fetchMock } as unknown as Parameters<typeof load>[0]);

  expect(data).toEqual(settings);
  expect(fetchMock).toHaveBeenCalledWith("/api/settings", {});
});

it("fills the form from the saved settings, never with the key", () => {
  stubFetch();
  renderSettings({ url: "http://localhost:5030", hasApiKey: true });

  const key = screen.getByLabelText("API key") as HTMLInputElement;

  expect((screen.getByLabelText("Address") as HTMLInputElement).value).toBe("http://localhost:5030");
  expect(key.value).toBe("");
  expect(key.placeholder).toBe("Saved; type a new one to replace it");
});

it("saves the address and a new key", async () => {
  const fetchMock = stubFetch();
  const user = userEvent.setup();

  renderSettings();

  await user.type(screen.getByLabelText("Address"), "http://nas.lan:5030 ");
  await user.type(screen.getByLabelText("API key"), "secret-key-0123456789");
  await user.click(screen.getByRole("button", { name: "Save" }));

  expect(await screen.findByText("Saved")).toBeTruthy();
  expect(saves(fetchMock)).toEqual([
    { slskd: { url: "http://nas.lan:5030", apiKey: "secret-key-0123456789" } },
  ]);

  const key = screen.getByLabelText("API key") as HTMLInputElement;

  expect(key.value).toBe("");
  expect(key.placeholder).toBe("Saved; type a new one to replace it");
});

it("keeps the saved key when its field is left blank", async () => {
  const fetchMock = stubFetch();
  const user = userEvent.setup();

  renderSettings({ url: "http://localhost:5030", hasApiKey: true });

  await user.clear(screen.getByLabelText("Address"));
  await user.type(screen.getByLabelText("Address"), "http://nas.lan:5030");
  await user.type(screen.getByLabelText("API key"), "   ");
  await user.click(screen.getByRole("button", { name: "Save" }));

  await screen.findByText("Saved");

  expect(saves(fetchMock)).toEqual([{ slskd: { url: "http://nas.lan:5030" } }]);
});

it("checks slskd again after a save", async () => {
  stubFetch({ statuses: [notSetUp, loggedIn] });
  const user = userEvent.setup();

  renderSettings();

  expect(await screen.findByText("Not set up yet.")).toBeTruthy();

  await user.type(screen.getByLabelText("Address"), "http://localhost:5030");
  await user.click(screen.getByRole("button", { name: "Save" }));

  expect(await screen.findByText("Logged in to Soulseek as guardarr.")).toBeTruthy();
});

it.each([
  [
    { configured: true, reachable: true, loggedIn: false },
    "slskd answers, but it is not logged in to Soulseek.",
  ],
  [
    { configured: true, reachable: false, loggedIn: false, problem: "slskd refused the API key." },
    "slskd refused the API key.",
  ],
])("says how slskd is doing", async (status, text) => {
  stubFetch({ statuses: [status] });
  renderSettings({ url: "http://localhost:5030", hasApiKey: true });

  expect(await screen.findByText(text)).toBeTruthy();
});

it("says when slskd could not be checked", async () => {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(null, { status: 500 })));
  renderSettings();

  expect(await screen.findByText("Could not check slskd.")).toBeTruthy();
});

it("shows why saving failed and keeps what was typed", async () => {
  stubFetch({ saveStatus: 500 });
  const user = userEvent.setup();

  renderSettings();

  await user.type(screen.getByLabelText("Address"), "http://localhost:5030");
  await user.type(screen.getByLabelText("API key"), "secret-key-0123456789");
  await user.click(screen.getByRole("button", { name: "Save" }));

  expect((await screen.findByRole("alert")).textContent).toBe("saving the settings failed (500)");
  expect((screen.getByLabelText("API key") as HTMLInputElement).value).toBe("secret-key-0123456789");
  expect(screen.queryByText("Saved")).toBeNull();
});
