import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  update: vi.fn(),
  merge: vi.fn(),
  search: vi.fn(),
  eventsOn: vi.fn(),
  clipboard: vi.fn(),
}));

vi.mock("../wailsjs/go/main/App", () => ({
  Status: vi.fn(),
  SelectImportFile: vi.fn(),
  Import: vi.fn(),
  PageMarkdown: vi.fn(),
  DeletePage: vi.fn(),
  Pages: vi.fn(),
  View: vi.fn(),
  BlockContent: vi.fn(),
  Block: vi.fn(),
  UpdateBlock: mocks.update,
  DeleteBlock: vi.fn(),
  MergeBlocks: mocks.merge,
  MoveBlock: vi.fn(),
  SplitBlock: vi.fn(),
  ReorderBlocks: vi.fn(),
  Search: mocks.search,
  ExportJSON: vi.fn(),
  ExportMarkdown: vi.fn(),
  CancelTask: vi.fn(),
  DownloadAssets: vi.fn(),
  ConfirmClose: vi.fn(),
  SetPageToolsVisible: vi.fn(),
}));

vi.mock("../wailsjs/go/models", () => ({
  document: {
    UpdateBlockRequest: class {
      constructor(values: object) {
        Object.assign(this, values);
      }
    },
    SearchRequest: class {
      constructor(values: object) {
        Object.assign(this, values);
      }
    },
    MarkdownRequest: class {
      constructor(values: object) {
        Object.assign(this, values);
      }
    },
  },
}));

vi.mock("../wailsjs/runtime/runtime", () => ({
  ClipboardSetText: mocks.clipboard,
  EventsOn: mocks.eventsOn,
}));

import { api } from "./api";

describe("Wails API boundary", () => {
  beforeEach(() => vi.clearAllMocks());

  it("sends complete block updates and the default merge separator", () => {
    api.block.update(2, 7, {
      label: "text",
      content: "updated content",
      bbox: "1,2,3,4",
      order: "5",
    });
    expect(mocks.update).toHaveBeenCalledWith({
      page: 2,
      block: 7,
      label: "text",
      content: "updated content",
      bbox: "1,2,3,4",
      order: "5",
    });

    api.block.merge(2, "7,8");
    expect(mocks.merge).toHaveBeenCalledWith(2, "7,8", "\n");
  });

  it("normalizes optional search fields before crossing the boundary", () => {
    api.search.query({
      label: "text",
      content: "needle",
      page_from: "0",
      page_to: "4",
    });
    expect(mocks.search).toHaveBeenCalledWith({
      label: "text",
      content: "needle",
      pageFrom: 0,
      pageTo: 4,
    });

    api.search.query({});
    expect(mocks.search).toHaveBeenLastCalledWith({
      label: null,
      content: null,
      pageFrom: null,
      pageTo: null,
    });
  });

  it("subscribes to the exact runtime event names", () => {
    const callback = vi.fn();
    api.task.onProgress(callback);
    api.app.onMenuCommand(callback);
    api.app.onCloseRequested(callback);
    expect(mocks.eventsOn.mock.calls.map(([name]) => name)).toEqual([
      "task:progress",
      "menu:command",
      "app:close-requested",
    ]);
  });
});
