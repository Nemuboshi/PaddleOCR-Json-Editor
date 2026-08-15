import * as App from "../wailsjs/go/main/App";
import { document as models } from "../wailsjs/go/models";
import { ClipboardSetText, EventsOn } from "../wailsjs/runtime/runtime";
import type { PageViewResponse } from "./types";

export interface TaskProgress {
  task: string;
  done: number;
  total: number;
  stage: string;
}

export const api = {
  document: {
    status: App.Status,
    selectImportFile: App.SelectImportFile,
    import: (path: string, downloadAssets: boolean) =>
      App.Import(path, downloadAssets),
  },
  page: {
    content: App.PageMarkdown,
    delete: App.DeletePage,
    list: (pageFrom = 0) => App.Pages(pageFrom),
    view: async (page: number) =>
      (await App.View(page)) as unknown as PageViewResponse,
  },
  block: {
    content: App.BlockContent,
    detail: App.Block,
    update: (
      page: number,
      block: number,
      body: { label: string; content: string; bbox: string; order: string },
    ) =>
      App.UpdateBlock(new models.UpdateBlockRequest({ page, block, ...body })),
    delete: App.DeleteBlock,
    merge: (page: number, blocks: string, separator = "\n") =>
      App.MergeBlocks(page, blocks, separator),
    move: App.MoveBlock,
    split: App.SplitBlock,
    reorder: App.ReorderBlocks,
  },
  search: {
    query: (params: Record<string, string>) =>
      App.Search(
        new models.SearchRequest({
          label: params.label || null,
          content: params.content || null,
          pageFrom: params.page_from ? Number(params.page_from) : null,
          pageTo: params.page_to ? Number(params.page_to) : null,
        }),
      ),
  },
  export: {
    json: App.ExportJSON,
    markdown: (start: number, end: number) =>
      App.ExportMarkdown(new models.MarkdownRequest({ start, end })),
  },
  task: {
    cancel: App.CancelTask,
    downloadAssets: App.DownloadAssets,
    onProgress: (callback: (progress: TaskProgress) => void) =>
      EventsOn("task:progress", callback),
  },
  clipboard: { writeText: ClipboardSetText },
  app: {
    confirmClose: App.ConfirmClose,
    pageToolsVisible: App.SetPageToolsVisible,
    onMenuCommand: (callback: (command: string) => void) =>
      EventsOn("menu:command", callback),
    onCloseRequested: (callback: () => void) =>
      EventsOn("app:close-requested", callback),
  },
};
