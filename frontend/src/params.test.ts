import { expect, it } from "vitest";
import { params } from "./params.ts";

function matches(param: string) {
  const result = params.mbid["~standard"].validate(param);

  return !(result instanceof Promise) && result.issues === undefined;
}

it("matches a MusicBrainz id", () => {
  expect(matches("5c9b2a1e-6f3d-4a8b-9e07-1d2c3b4a5f60")).toBe(true);
});

it.each(["abc", "not-an-id", "5C9B2A1E-6F3D-4A8B-9E07-1D2C3B4A5F60", "5c9b2a1e-6f3d-4a8b-9e07-1d2c3b4a5f60x"])(
  "turns down %s",
  (param) => {
    expect(matches(param)).toBe(false);
  },
);
