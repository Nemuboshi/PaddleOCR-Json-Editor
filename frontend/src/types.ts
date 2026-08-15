import type { document } from "../wailsjs/go/models";

export type StatusResponse = document.Status;
export type PageSummary = document.PageSummary;
export type PageDetailResponse = document.PageDetail;
export type BlockDetailResponse = document.BlockDetail;
export interface LayoutBlockJson extends Omit<
  document.LayoutBlock,
  "bbox" | "order"
> {
  bbox: [number, number, number, number];
  order: number | null;
}
export interface PageLayoutJson extends Omit<
  document.PageLayout,
  "blocks" | "input_image" | "boxed_image"
> {
  input_image: string | null;
  boxed_image: string | null;
  blocks: LayoutBlockJson[];
}
export interface PageViewResponse {
  detail: PageDetailResponse;
  layout: PageLayoutJson;
}
export type MessageResponse = document.Message;
export type QueryHit = document.SearchHit;
