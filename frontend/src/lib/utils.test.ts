import { describe, expect, it } from "vitest";
import { compactFileName } from "./utils";

describe("compactFileName", () => {
  it("shows only the filename and preserves both ends", () => {
    expect(compactFileName("C:\\very\\long\\report.json")).toBe("report.json");
    expect(compactFileName("C:\\docs\\abcdefghijklmnop.json", 13)).toBe(
      "abcdef…p.json",
    );
  });
});
