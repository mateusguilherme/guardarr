import { getContext, setContext } from "svelte";
import { SvelteMap, SvelteSet } from "svelte/reactivity";
import {
  followArtist,
  followedArtists,
  searchArtists,
  unfollowArtist,
  type Artist,
} from "./api";

export type Search =
  | { status: "idle" }
  | { status: "searching" }
  | { status: "done"; artists: Artist[] }
  | { status: "error"; message: string };

// AppState holds what outlives a page: the followed artists and the latest
// search, so coming back from an artist finds the home as the user left it.
// The root layout creates it once and the pages read it from the context.
export class AppState {
  readonly followed = new SvelteMap<string, Artist>();
  readonly pending = new SvelteSet<string>();
  // Settles, loaded or failed, once; until then the artist page cannot
  // tell a followed artist from one it has to look up.
  followedLoaded = $state(false);

  term = $state("");
  search = $state<Search>({ status: "idle" });
  // Bumped on reset so the result of an in-flight search is discarded
  // instead of reappearing over the home.
  #searchSeq = 0;

  constructor() {
    this.#loadFollowed();
  }

  async #loadFollowed() {
    try {
      for (const artist of await followedArtists()) {
        this.followed.set(artist.id, artist);
      }
    } catch {
      // only means an empty home and every button starting as "Follow"
    } finally {
      this.followedLoaded = true;
    }
  }

  async follow(artist: Artist) {
    if (this.followed.has(artist.id) || this.pending.has(artist.id)) {
      return;
    }

    this.pending.add(artist.id);

    try {
      await followArtist(artist);
      this.followed.set(artist.id, artist);
    } catch {
      // keep the button enabled so the user can retry
    } finally {
      this.pending.delete(artist.id);
    }
  }

  async unfollow(id: string) {
    if (!this.followed.has(id) || this.pending.has(id)) {
      return;
    }

    this.pending.add(id);

    try {
      await unfollowArtist(id);
      this.followed.delete(id);
    } catch {
      // still following; the button stays enabled so the user can retry
    } finally {
      this.pending.delete(id);
    }
  }

  // runSearch looks the term up on MusicBrainz, unless it is blank or a
  // search is already running.
  async runSearch() {
    const q = this.term.trim();

    if (!q || this.search.status === "searching") {
      return;
    }

    const seq = ++this.#searchSeq;

    this.search = { status: "searching" };

    try {
      const artists = await searchArtists(q);

      if (seq === this.#searchSeq) {
        this.search = { status: "done", artists };
      }
    } catch (err) {
      if (seq === this.#searchSeq) {
        this.search = {
          status: "error",
          message: err instanceof Error ? err.message : String(err),
        };
      }
    }
  }

  resetSearch() {
    this.#searchSeq++;
    this.search = { status: "idle" };
  }

  // The artist page usually gets its name from what the home already holds
  // — the followed map or the latest search results — so opening it costs
  // no extra lookup.
  knownArtist(id: string): Artist | undefined {
    return (
      this.followed.get(id) ??
      (this.search.status === "done"
        ? this.search.artists.find((artist) => artist.id === id)
        : undefined)
    );
  }
}

const key = Symbol("app state");

export function setAppState(state: AppState) {
  setContext(key, state);
}

export function getAppState(): AppState {
  return getContext(key);
}

// appStateContext lets a test mount a page on its own, with the context the
// root layout would give it.
export function appStateContext(state: AppState) {
  // eslint-disable-next-line svelte/prefer-svelte-reactivity -- mount reads it once; nothing watches it
  return new Map([[key, state]]);
}
