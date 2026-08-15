export namespace document {
	
	export class BlockDetail {
	    page_index: number;
	    block_id: number;
	    label: string;
	    content: string;
	    bbox: string;
	    order: string;
	
	    static createFrom(source: any = {}) {
	        return new BlockDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.page_index = source["page_index"];
	        this.block_id = source["block_id"];
	        this.label = source["label"];
	        this.content = source["content"];
	        this.bbox = source["bbox"];
	        this.order = source["order"];
	    }
	}
	export class ImportResult {
	    total_pages: number;
	    total_blocks: number;
	    downloaded: number;
	    failed: number;
	
	    static createFrom(source: any = {}) {
	        return new ImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_pages = source["total_pages"];
	        this.total_blocks = source["total_blocks"];
	        this.downloaded = source["downloaded"];
	        this.failed = source["failed"];
	    }
	}
	export class LayoutBlock {
	    block_id: number;
	    label: string;
	    bbox: number[];
	    content: string;
	    order?: number;
	
	    static createFrom(source: any = {}) {
	        return new LayoutBlock(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.block_id = source["block_id"];
	        this.label = source["label"];
	        this.bbox = source["bbox"];
	        this.content = source["content"];
	        this.order = source["order"];
	    }
	}
	export class MarkdownRequest {
	    start: number;
	    end: number;
	
	    static createFrom(source: any = {}) {
	        return new MarkdownRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.start = source["start"];
	        this.end = source["end"];
	    }
	}
	export class Status {
	    loaded: boolean;
	    changed: boolean;
	    total_pages: number;
	    total_blocks: number;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.loaded = source["loaded"];
	        this.changed = source["changed"];
	        this.total_pages = source["total_pages"];
	        this.total_blocks = source["total_blocks"];
	        this.source = source["source"];
	    }
	}
	export class PageBlockRow {
	    block_id: number;
	    label: string;
	    preview: string;
	    order: string;
	    bbox: string;
	
	    static createFrom(source: any = {}) {
	        return new PageBlockRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.block_id = source["block_id"];
	        this.label = source["label"];
	        this.preview = source["preview"];
	        this.order = source["order"];
	        this.bbox = source["bbox"];
	    }
	}
	export class PageDetail {
	    page_index: number;
	    block_count: number;
	    blocks: PageBlockRow[];
	    image_url?: string;
	    input_image: string;
	    is_remote_image: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PageDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.page_index = source["page_index"];
	        this.block_count = source["block_count"];
	        this.blocks = this.convertValues(source["blocks"], PageBlockRow);
	        this.image_url = source["image_url"];
	        this.input_image = source["input_image"];
	        this.is_remote_image = source["is_remote_image"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Message {
	    message: string;
	    page?: PageDetail;
	    status?: Status;
	
	    static createFrom(source: any = {}) {
	        return new Message(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.message = source["message"];
	        this.page = this.convertValues(source["page"], PageDetail);
	        this.status = this.convertValues(source["status"], Status);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class PageLayout {
	    input_image?: string;
	    boxed_image?: string;
	    images: Record<string, string>;
	    blocks: LayoutBlock[];
	
	    static createFrom(source: any = {}) {
	        return new PageLayout(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.input_image = source["input_image"];
	        this.boxed_image = source["boxed_image"];
	        this.images = source["images"];
	        this.blocks = this.convertValues(source["blocks"], LayoutBlock);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PageSummary {
	    index: number;
	    block_count: number;
	    labels_summary: string;
	
	    static createFrom(source: any = {}) {
	        return new PageSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.block_count = source["block_count"];
	        this.labels_summary = source["labels_summary"];
	    }
	}
	export class PageView {
	    detail: PageDetail;
	    layout: PageLayout;
	
	    static createFrom(source: any = {}) {
	        return new PageView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.detail = this.convertValues(source["detail"], PageDetail);
	        this.layout = this.convertValues(source["layout"], PageLayout);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PagesResponse {
	    pages: PageSummary[];
	    page_from: number;
	    page_to: number;
	    total_pages: number;
	
	    static createFrom(source: any = {}) {
	        return new PagesResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pages = this.convertValues(source["pages"], PageSummary);
	        this.page_from = source["page_from"];
	        this.page_to = source["page_to"];
	        this.total_pages = source["total_pages"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SearchHit {
	    page_index: number;
	    block_id: number;
	    label: string;
	    preview: string;
	
	    static createFrom(source: any = {}) {
	        return new SearchHit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.page_index = source["page_index"];
	        this.block_id = source["block_id"];
	        this.label = source["label"];
	        this.preview = source["preview"];
	    }
	}
	export class SearchRequest {
	    label?: string;
	    content?: string;
	    pageFrom?: number;
	    pageTo?: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.content = source["content"];
	        this.pageFrom = source["pageFrom"];
	        this.pageTo = source["pageTo"];
	    }
	}
	export class SearchResponse {
	    hits: SearchHit[];
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hits = this.convertValues(source["hits"], SearchHit);
	        this.total = source["total"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class UpdateBlockRequest {
	    page: number;
	    block: number;
	    label: string;
	    content: string;
	    bbox: string;
	    order: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateBlockRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.page = source["page"];
	        this.block = source["block"];
	        this.label = source["label"];
	        this.content = source["content"];
	        this.bbox = source["bbox"];
	        this.order = source["order"];
	    }
	}

}

