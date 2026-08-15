/** Screen-space point used to place the floating block editor. */
export type EditorAnchor = {
  x: number;
  y: number;
  leftEdge?: number;
};

export function anchorFromRect(
  rect: DOMRect,
  prefer: "right" | "below" = "right",
): EditorAnchor {
  if (prefer === "below") {
    return { x: rect.left, y: rect.bottom + 8 };
  }
  return { x: rect.right + 8, y: rect.top, leftEdge: rect.left - 8 };
}

export function viewportCenterAnchor(): EditorAnchor {
  return {
    x: Math.round(window.innerWidth * 0.55),
    y: Math.round(window.innerHeight * 0.2),
  };
}
