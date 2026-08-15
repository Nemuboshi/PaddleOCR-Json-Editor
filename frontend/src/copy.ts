export type ErrorAction = "import" | "retry" | "fix_bbox" | "pick_block" | null;

export const formatErrorText = (
  message: string,
  hint: string | null,
): string => (hint ? `${message} — ${hint}` : message);

export const errorText: Record<string, { hint: string; action: ErrorAction }> =
  {
    not_loaded: { hint: "Import a Paddle JSON file first.", action: "import" },
    file_not_found: {
      hint: "Make sure that the file path is correct and the file exists.",
      action: null,
    },
    json_error: {
      hint: "Expected a PP-StructureV3 JSON array. Choose another file.",
      action: "import",
    },
    page_not_found: {
      hint: "The page or its index is no longer available. Reload the page list.",
      action: "retry",
    },
    block_not_found: {
      hint: "The block is no longer available. Select another block.",
      action: "pick_block",
    },
    invalid_bbox: {
      hint: "Use four comma-separated numbers for the bounding box: x1,y1,x2,y2.",
      action: "fix_bbox",
    },
    merge_too_few: {
      hint: "Enter at least two block IDs to merge.",
      action: null,
    },
    split_out_of_range: {
      hint: "Set the split position within the text.",
      action: null,
    },
    reorder_mismatch: {
      hint: "Include every block ID on the current page.",
      action: null,
    },
  };
