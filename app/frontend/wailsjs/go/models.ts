export namespace main {
	
	export class SubItem {
	    mat_id: string;
	    name: string;
	    bags: number;
	
	    static createFrom(source: any = {}) {
	        return new SubItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mat_id = source["mat_id"];
	        this.name = source["name"];
	        this.bags = source["bags"];
	    }
	}
	export class ComputeInput {
	    sub_items: SubItem[];
	    platform_price: number;
	    monthly_revenue_target: number;
	    target_mode: string;
	    revenue_share: number;
	    ap_rate: number;
	    mp_rate: number;
	    traffic_cost: number;
	    tp12_rate?: number;
	    cp52_rate: number;
	    cp52_slot: number;
	    cp53_rate: number;
	    logistics_cost: number;
	    fp_rate: number;
	
	    static createFrom(source: any = {}) {
	        return new ComputeInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sub_items = this.convertValues(source["sub_items"], SubItem);
	        this.platform_price = source["platform_price"];
	        this.monthly_revenue_target = source["monthly_revenue_target"];
	        this.target_mode = source["target_mode"];
	        this.revenue_share = source["revenue_share"];
	        this.ap_rate = source["ap_rate"];
	        this.mp_rate = source["mp_rate"];
	        this.traffic_cost = source["traffic_cost"];
	        this.tp12_rate = source["tp12_rate"];
	        this.cp52_rate = source["cp52_rate"];
	        this.cp52_slot = source["cp52_slot"];
	        this.cp53_rate = source["cp53_rate"];
	        this.logistics_cost = source["logistics_cost"];
	        this.fp_rate = source["fp_rate"];
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
	export class ComputeResult {
	    est_qty: number;
	    platform_rev_untax: number;
	    platform_rev_gsv: number;
	    company_rev: number;
	    std_cost_total: number;
	    gross_profit: number;
	    gross_rate: number;
	    ap: number;
	    ap_rate: number;
	    tp12: number;
	    tp22: number;
	    tp_total: number;
	    cp12: number;
	    cp51: number;
	    cp52: number;
	    cp53: number;
	    cp_total: number;
	    pp: number;
	    mp: number;
	    mp_rate: number;
	    marketing_fee: number;
	    assess_net: number;
	    fp: number;
	    ae: number;
	    non_support: number;
	    op_net: number;
	    discount_rate: number;
	    aov: number;
	
	    static createFrom(source: any = {}) {
	        return new ComputeResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.est_qty = source["est_qty"];
	        this.platform_rev_untax = source["platform_rev_untax"];
	        this.platform_rev_gsv = source["platform_rev_gsv"];
	        this.company_rev = source["company_rev"];
	        this.std_cost_total = source["std_cost_total"];
	        this.gross_profit = source["gross_profit"];
	        this.gross_rate = source["gross_rate"];
	        this.ap = source["ap"];
	        this.ap_rate = source["ap_rate"];
	        this.tp12 = source["tp12"];
	        this.tp22 = source["tp22"];
	        this.tp_total = source["tp_total"];
	        this.cp12 = source["cp12"];
	        this.cp51 = source["cp51"];
	        this.cp52 = source["cp52"];
	        this.cp53 = source["cp53"];
	        this.cp_total = source["cp_total"];
	        this.pp = source["pp"];
	        this.mp = source["mp"];
	        this.mp_rate = source["mp_rate"];
	        this.marketing_fee = source["marketing_fee"];
	        this.assess_net = source["assess_net"];
	        this.fp = source["fp"];
	        this.ae = source["ae"];
	        this.non_support = source["non_support"];
	        this.op_net = source["op_net"];
	        this.discount_rate = source["discount_rate"];
	        this.aov = source["aov"];
	    }
	}
	export class ConfigRow {
	    year_month: string;
	    department: string;
	    channel: string;
	    mode: string;
	    shop: string;
	    monthly_revenue_target: number;
	    target_mode: string;
	    combo_name: string;
	    revenue_share: number;
	    sub_items: SubItem[];
	    platform_price: number;
	    ap_rate: number;
	    mp_rate: number;
	    traffic_cost: number;
	    tp12_rate?: number;
	    cp52_rate: number;
	    cp52_slot: number;
	    cp53_rate: number;
	    logistics_cost: number;
	    fp_rate: number;
	
	    static createFrom(source: any = {}) {
	        return new ConfigRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.year_month = source["year_month"];
	        this.department = source["department"];
	        this.channel = source["channel"];
	        this.mode = source["mode"];
	        this.shop = source["shop"];
	        this.monthly_revenue_target = source["monthly_revenue_target"];
	        this.target_mode = source["target_mode"];
	        this.combo_name = source["combo_name"];
	        this.revenue_share = source["revenue_share"];
	        this.sub_items = this.convertValues(source["sub_items"], SubItem);
	        this.platform_price = source["platform_price"];
	        this.ap_rate = source["ap_rate"];
	        this.mp_rate = source["mp_rate"];
	        this.traffic_cost = source["traffic_cost"];
	        this.tp12_rate = source["tp12_rate"];
	        this.cp52_rate = source["cp52_rate"];
	        this.cp52_slot = source["cp52_slot"];
	        this.cp53_rate = source["cp53_rate"];
	        this.logistics_cost = source["logistics_cost"];
	        this.fp_rate = source["fp_rate"];
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
	export class DimensionInfo {
	    department: string;
	    channel: string;
	    mode: string;
	    shop: string;
	    monthly_revenue_target: number;
	    target_mode: string;
	
	    static createFrom(source: any = {}) {
	        return new DimensionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.department = source["department"];
	        this.channel = source["channel"];
	        this.mode = source["mode"];
	        this.shop = source["shop"];
	        this.monthly_revenue_target = source["monthly_revenue_target"];
	        this.target_mode = source["target_mode"];
	    }
	}
	export class FillPivotAutoRequest {
	    department: string;
	    overwrite: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FillPivotAutoRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.department = source["department"];
	        this.overwrite = source["overwrite"];
	    }
	}
	export class PivotFilledBlock {
	    block: string;
	    cell_count: number;
	
	    static createFrom(source: any = {}) {
	        return new PivotFilledBlock(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.block = source["block"];
	        this.cell_count = source["cell_count"];
	    }
	}
	export class PivotStats {
	    blocks: number;
	    months: number;
	    cells: number;
	
	    static createFrom(source: any = {}) {
	        return new PivotStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.blocks = source["blocks"];
	        this.months = source["months"];
	        this.cells = source["cells"];
	    }
	}
	export class FillPivotAutoResult {
	    output_path: string;
	    department: string;
	    stats: PivotStats;
	    filled: PivotFilledBlock[];
	
	    static createFrom(source: any = {}) {
	        return new FillPivotAutoResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.output_path = source["output_path"];
	        this.department = source["department"];
	        this.stats = this.convertValues(source["stats"], PivotStats);
	        this.filled = this.convertValues(source["filled"], PivotFilledBlock);
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
	export class PivotFillItem {
	    block: string;
	    year_month: string;
	    fee_item: string;
	    wan: number;
	    rate: number;
	    has_rate: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PivotFillItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.block = source["block"];
	        this.year_month = source["year_month"];
	        this.fee_item = source["fee_item"];
	        this.wan = source["wan"];
	        this.rate = source["rate"];
	        this.has_rate = source["has_rate"];
	    }
	}
	export class FillPivotRequest {
	    file_path: string;
	    items: PivotFillItem[];
	
	    static createFrom(source: any = {}) {
	        return new FillPivotRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_path = source["file_path"];
	        this.items = this.convertValues(source["items"], PivotFillItem);
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
	export class FillRecord {
	    department: string;
	    channel: string;
	    mode: string;
	    shop: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new FillRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.department = source["department"];
	        this.channel = source["channel"];
	        this.mode = source["mode"];
	        this.shop = source["shop"];
	        this.reason = source["reason"];
	    }
	}
	export class FillRequest {
	    file_path: string;
	    year_month: string;
	    new_data: ConfigRow[];
	    overwrite_map: Record<string, boolean>;
	
	    static createFrom(source: any = {}) {
	        return new FillRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_path = source["file_path"];
	        this.year_month = source["year_month"];
	        this.new_data = this.convertValues(source["new_data"], ConfigRow);
	        this.overwrite_map = source["overwrite_map"];
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
	export class FillSplitAutoRequest {
	    year_month: string;
	    overwrite: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FillSplitAutoRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.year_month = source["year_month"];
	        this.overwrite = source["overwrite"];
	    }
	}
	export class FillStats {
	    total_positions: number;
	    config_found: number;
	    no_config: number;
	    already_filled: number;
	
	    static createFrom(source: any = {}) {
	        return new FillStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_positions = source["total_positions"];
	        this.config_found = source["config_found"];
	        this.no_config = source["no_config"];
	        this.already_filled = source["already_filled"];
	    }
	}
	export class FillSplitAutoResult {
	    output_path: string;
	    year_month: string;
	    filled: FillRecord[];
	    skipped: FillRecord[];
	    stats: FillStats;
	
	    static createFrom(source: any = {}) {
	        return new FillSplitAutoResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.output_path = source["output_path"];
	        this.year_month = source["year_month"];
	        this.filled = this.convertValues(source["filled"], FillRecord);
	        this.skipped = this.convertValues(source["skipped"], FillRecord);
	        this.stats = this.convertValues(source["stats"], FillStats);
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
	
	export class JieAnExportRow {
	    ax_code: string;
	    revenue: number;
	    creator_comm: number;
	    own_revenue: number;
	    merged_revenue: number;
	    count: number;
	    unit_price: number;
	    is_merged: boolean;
	    shop: string;
	    platform_price: number;
	    creator_rate: number;
	    creator_slot: number;
	    leader_rate: number;
	    logistics: number;
	    ap_rate: number;
	    mp_rate: number;
	    mat_ids: string[];
	    sub_items: SubItem[];
	
	    static createFrom(source: any = {}) {
	        return new JieAnExportRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ax_code = source["ax_code"];
	        this.revenue = source["revenue"];
	        this.creator_comm = source["creator_comm"];
	        this.own_revenue = source["own_revenue"];
	        this.merged_revenue = source["merged_revenue"];
	        this.count = source["count"];
	        this.unit_price = source["unit_price"];
	        this.is_merged = source["is_merged"];
	        this.shop = source["shop"];
	        this.platform_price = source["platform_price"];
	        this.creator_rate = source["creator_rate"];
	        this.creator_slot = source["creator_slot"];
	        this.leader_rate = source["leader_rate"];
	        this.logistics = source["logistics"];
	        this.ap_rate = source["ap_rate"];
	        this.mp_rate = source["mp_rate"];
	        this.mat_ids = source["mat_ids"];
	        this.sub_items = this.convertValues(source["sub_items"], SubItem);
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
	export class JieAnExportRequest {
	    channel: string;
	    rows: JieAnExportRow[];
	
	    static createFrom(source: any = {}) {
	        return new JieAnExportRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.channel = source["channel"];
	        this.rows = this.convertValues(source["rows"], JieAnExportRow);
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
	
	export class JieAnSKU {
	    ax_code: string;
	    display_name: string;
	    count: number;
	    revenue: number;
	    creator_comm: number;
	    own_revenue: number;
	    merged_revenue: number;
	    creator_rate: number;
	    platform_price: number;
	    unit_price: number;
	    mat_ids: string[];
	    mat_names: string[];
	    sub_items: SubItem[];
	    is_merged: boolean;
	    merged_from: string[];
	
	    static createFrom(source: any = {}) {
	        return new JieAnSKU(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ax_code = source["ax_code"];
	        this.display_name = source["display_name"];
	        this.count = source["count"];
	        this.revenue = source["revenue"];
	        this.creator_comm = source["creator_comm"];
	        this.own_revenue = source["own_revenue"];
	        this.merged_revenue = source["merged_revenue"];
	        this.creator_rate = source["creator_rate"];
	        this.platform_price = source["platform_price"];
	        this.unit_price = source["unit_price"];
	        this.mat_ids = source["mat_ids"];
	        this.mat_names = source["mat_names"];
	        this.sub_items = this.convertValues(source["sub_items"], SubItem);
	        this.is_merged = source["is_merged"];
	        this.merged_from = source["merged_from"];
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
	export class JieAnResult {
	    skus: JieAnSKU[];
	    threshold: number;
	    total_rows: number;
	    channel: string;
	
	    static createFrom(source: any = {}) {
	        return new JieAnResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.skus = this.convertValues(source["skus"], JieAnSKU);
	        this.threshold = source["threshold"];
	        this.total_rows = source["total_rows"];
	        this.channel = source["channel"];
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
	
	export class MasterUpdateRequest {
	    file_path: string;
	    product_list: boolean;
	    std_cost: boolean;
	    assess_rev: boolean;
	    dist_price: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MasterUpdateRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_path = source["file_path"];
	        this.product_list = source["product_list"];
	        this.std_cost = source["std_cost"];
	        this.assess_rev = source["assess_rev"];
	        this.dist_price = source["dist_price"];
	    }
	}
	export class MasterUpdateResult {
	    ok: boolean;
	    message: string;
	    counts: Record<string, number>;
	    created: number;
	    updated: number;
	
	    static createFrom(source: any = {}) {
	        return new MasterUpdateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.message = source["message"];
	        this.counts = source["counts"];
	        this.created = source["created"];
	        this.updated = source["updated"];
	    }
	}
	export class MaterialSearchResult {
	    mat_id: string;
	    name: string;
	    std_cost?: number;
	    ec_assess_revenue?: number;
	    retail_spec: number;
	
	    static createFrom(source: any = {}) {
	        return new MaterialSearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mat_id = source["mat_id"];
	        this.name = source["name"];
	        this.std_cost = source["std_cost"];
	        this.ec_assess_revenue = source["ec_assess_revenue"];
	        this.retail_spec = source["retail_spec"];
	    }
	}
	export class PivotCell {
	    block: string;
	    year_month: string;
	    fee_item: string;
	    wan: number;
	    has_rate: boolean;
	    rate: number;
	    excel_row: number;
	    amount_col: number;
	    rate_col: number;
	
	    static createFrom(source: any = {}) {
	        return new PivotCell(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.block = source["block"];
	        this.year_month = source["year_month"];
	        this.fee_item = source["fee_item"];
	        this.wan = source["wan"];
	        this.has_rate = source["has_rate"];
	        this.rate = source["rate"];
	        this.excel_row = source["excel_row"];
	        this.amount_col = source["amount_col"];
	        this.rate_col = source["rate_col"];
	    }
	}
	
	
	export class PivotScanResult {
	    blocks: string[];
	    months: string[];
	    fee_items: string[];
	    cells: PivotCell[];
	    block_rows: Record<string, number>;
	
	    static createFrom(source: any = {}) {
	        return new PivotScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.blocks = source["blocks"];
	        this.months = source["months"];
	        this.fee_items = source["fee_items"];
	        this.cells = this.convertValues(source["cells"], PivotCell);
	        this.block_rows = source["block_rows"];
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
	
	export class ProcessOrderFilesReq {
	    channel: string;
	    file_paths: string[];
	    threshold: number;
	    password: string;
	
	    static createFrom(source: any = {}) {
	        return new ProcessOrderFilesReq(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.channel = source["channel"];
	        this.file_paths = source["file_paths"];
	        this.threshold = source["threshold"];
	        this.password = source["password"];
	    }
	}
	export class SaveConfigRequest {
	    year_month: string;
	    department: string;
	    channel: string;
	    mode: string;
	    shop: string;
	    monthly_revenue_target: number;
	    target_mode: string;
	    rows: ConfigRow[];
	
	    static createFrom(source: any = {}) {
	        return new SaveConfigRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.year_month = source["year_month"];
	        this.department = source["department"];
	        this.channel = source["channel"];
	        this.mode = source["mode"];
	        this.shop = source["shop"];
	        this.monthly_revenue_target = source["monthly_revenue_target"];
	        this.target_mode = source["target_mode"];
	        this.rows = this.convertValues(source["rows"], ConfigRow);
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
	export class ScanConfigItem {
	    year_month: string;
	    department: string;
	    channel: string;
	    mode: string;
	    shop: string;
	    has_data: boolean;
	    sku_count: number;
	    excel_row: number;
	    sku_names: string[];
	
	    static createFrom(source: any = {}) {
	        return new ScanConfigItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.year_month = source["year_month"];
	        this.department = source["department"];
	        this.channel = source["channel"];
	        this.mode = source["mode"];
	        this.shop = source["shop"];
	        this.has_data = source["has_data"];
	        this.sku_count = source["sku_count"];
	        this.excel_row = source["excel_row"];
	        this.sku_names = source["sku_names"];
	    }
	}
	export class ScanResult {
	    year_month: string;
	    shops_by_channel: Record<string, Array<string>>;
	    configs: ScanConfigItem[];
	
	    static createFrom(source: any = {}) {
	        return new ScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.year_month = source["year_month"];
	        this.shops_by_channel = source["shops_by_channel"];
	        this.configs = this.convertValues(source["configs"], ScanConfigItem);
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
	export class SessionState {
	    mode: string;
	    split_file_path: string;
	    pivot_file_path: string;
	    imported_shops: Record<string, Array<string>>;
	    imported_modes: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new SessionState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.split_file_path = source["split_file_path"];
	        this.pivot_file_path = source["pivot_file_path"];
	        this.imported_shops = source["imported_shops"];
	        this.imported_modes = source["imported_modes"];
	    }
	}
	export class SkuLibraryItem {
	    sku_name: string;
	    sub_items: SubItem[];
	
	    static createFrom(source: any = {}) {
	        return new SkuLibraryItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sku_name = source["sku_name"];
	        this.sub_items = this.convertValues(source["sub_items"], SubItem);
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
	
	export class UpdateB2GResult {
	    ok: boolean;
	    message: string;
	    rows: number;
	    combos: number;
	    added: number;
	    replaced: number;
	
	    static createFrom(source: any = {}) {
	        return new UpdateB2GResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.message = source["message"];
	        this.rows = source["rows"];
	        this.combos = source["combos"];
	        this.added = source["added"];
	        this.replaced = source["replaced"];
	    }
	}
	export class ZhouBaoExportRequest {
	    channel: string;
	    rows: JieAnExportRow[];
	
	    static createFrom(source: any = {}) {
	        return new ZhouBaoExportRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.channel = source["channel"];
	        this.rows = this.convertValues(source["rows"], JieAnExportRow);
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

}

