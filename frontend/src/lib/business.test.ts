import { beforeEach, describe, expect, it, vi } from "vitest";
import { formatErrorText } from "../copy";
import { anchorFromRect, viewportCenterAnchor } from "./anchor";
import { presentCommandError } from "./errors";
import { clamp, compactFileName, stripHtml } from "./utils";

describe("command errors", () => {
  it("maps Wails JSON errors to a useful recovery action", () => {
    expect(
      presentCommandError(
        new Error('invoke failed: {"code":"invalid_bbox","message":"Bad box"}'),
      ),
    ).toEqual({
      code: "invalid_bbox",
      message: "Bad box",
      hint: "Use four comma-separated numbers for the bounding box: x1,y1,x2,y2.",
      action: "fix_bbox",
    });
  });

  it("accepts structured and unknown errors without hiding their message", () => {
    expect(
      presentCommandError({ code: "block_not_found", message: "Gone" }),
    ).toMatchObject({ code: "block_not_found", action: "pick_block" });
    expect(presentCommandError(new Error("offline"))).toEqual({
      code: null,
      message: "offline",
      hint: null,
      action: null,
    });
    expect(formatErrorText("offline", "Retry later")).toBe(
      "offline — Retry later",
    );
  });
});

describe("display helpers", () => {
  beforeEach(() => {
    vi.stubGlobal("document", {
      createElement: () => ({
        set innerHTML(value: string) {
          this.innerText = value
            .replace(/<[^>]*>/g, " ")
            .replace(/&amp;/g, "&");
        },
        innerText: "",
      }),
    });
  });

  it("normalizes previews and keeps Unicode filenames intact", () => {
    expect(stripHtml("<b>Hello</b>\n <i>world</i>")).toBe("Hello world");
    expect(compactFileName("C:\\files\\alphabetomega.json", 10)).toBe(
      "alpha…json",
    );
    expect(clamp(120, 25, 100)).toBe(100);
  });
});

describe("editor placement", () => {
  it("anchors beside or below a selected block", () => {
    const rect = { left: 10, right: 110, top: 20, bottom: 70 } as DOMRect;
    expect(anchorFromRect(rect)).toEqual({ x: 118, y: 20, leftEdge: 2 });
    expect(anchorFromRect(rect, "below")).toEqual({ x: 10, y: 78 });
  });

  it("uses the viewport when no block is selected", () => {
    vi.stubGlobal("window", { innerWidth: 1000, innerHeight: 800 });
    expect(viewportCenterAnchor()).toEqual({ x: 550, y: 160 });
  });
});
