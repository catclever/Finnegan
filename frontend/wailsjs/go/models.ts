export namespace api {
	
	export class RadarResponse {
	    status: string;
	    results: string[];
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new RadarResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.results = source["results"];
	        this.error = source["error"];
	    }
	}

}

export namespace chatlog {
	
	export class Message {
	    speaker: string;
	    // Go type: time
	    timestamp?: any;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new Message(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.speaker = source["speaker"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.content = source["content"];
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
	export class ParsedChatLog {
	    userTitle?: string;
	    participants: string[];
	    // Go type: time
	    startedAt?: any;
	    // Go type: time
	    endedAt?: any;
	    messages: Message[];
	    coalesced: string;
	    images?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ParsedChatLog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.userTitle = source["userTitle"];
	        this.participants = source["participants"];
	        this.startedAt = this.convertValues(source["startedAt"], null);
	        this.endedAt = this.convertValues(source["endedAt"], null);
	        this.messages = this.convertValues(source["messages"], Message);
	        this.coalesced = source["coalesced"];
	        this.images = source["images"];
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
	export class TitleResult {
	    title: string;
	    participants: string[];
	
	    static createFrom(source: any = {}) {
	        return new TitleResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.participants = source["participants"];
	    }
	}

}

export namespace keywords {
	
	export class Config {
	    enabled_dimensions: string[];
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled_dimensions = source["enabled_dimensions"];
	    }
	}
	export class KeywordEngine {
	    LLMConfig: module.LLMConfig;
	    Config: Config;
	
	    static createFrom(source: any = {}) {
	        return new KeywordEngine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.LLMConfig = this.convertValues(source["LLMConfig"], module.LLMConfig);
	        this.Config = this.convertValues(source["Config"], Config);
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

export namespace main {
	
	export class Card {
	    id: string;
	    type: string;
	    origin?: string;
	    title: string;
	    content: string;
	    images?: string[];
	    sourceUrl?: string;
	    originalText?: string;
	    translatedText?: string;
	    publishedAt?: string;
	    authorName?: string;
	    authorAvatar?: string;
	    echoOf?: string;
	    matchedAnchor?: string;
	    echoReason?: string;
	    userComment?: string;
	    replyCount?: number;
	    repostCount?: number;
	    likeCount?: number;
	    // Go type: time
	    createdAt: any;
	    participants?: string[];
	    // Go type: time
	    startedAt?: any;
	    // Go type: time
	    endedAt?: any;
	    filePath?: string;
	    keywords?: Record<string, Array<string>>;
	    tags?: string[];
	    brand?: string;
	    platform?: string;
	    price?: string;
	    intent?: string;
	    availability?: string;
	    snapshotPath?: string;
	    screenshotPath?: string;
	
	    static createFrom(source: any = {}) {
	        return new Card(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.origin = source["origin"];
	        this.title = source["title"];
	        this.content = source["content"];
	        this.images = source["images"];
	        this.sourceUrl = source["sourceUrl"];
	        this.originalText = source["originalText"];
	        this.translatedText = source["translatedText"];
	        this.publishedAt = source["publishedAt"];
	        this.authorName = source["authorName"];
	        this.authorAvatar = source["authorAvatar"];
	        this.echoOf = source["echoOf"];
	        this.matchedAnchor = source["matchedAnchor"];
	        this.echoReason = source["echoReason"];
	        this.userComment = source["userComment"];
	        this.replyCount = source["replyCount"];
	        this.repostCount = source["repostCount"];
	        this.likeCount = source["likeCount"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.participants = source["participants"];
	        this.startedAt = this.convertValues(source["startedAt"], null);
	        this.endedAt = this.convertValues(source["endedAt"], null);
	        this.filePath = source["filePath"];
	        this.keywords = source["keywords"];
	        this.tags = source["tags"];
	        this.brand = source["brand"];
	        this.platform = source["platform"];
	        this.price = source["price"];
	        this.intent = source["intent"];
	        this.availability = source["availability"];
	        this.snapshotPath = source["snapshotPath"];
	        this.screenshotPath = source["screenshotPath"];
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
	export class ChatLogCaptureResult {
	    docId: string;
	    title: string;
	    participants?: string[];
	    // Go type: time
	    startedAt?: any;
	    // Go type: time
	    endedAt?: any;
	    images?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ChatLogCaptureResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.docId = source["docId"];
	        this.title = source["title"];
	        this.participants = source["participants"];
	        this.startedAt = this.convertValues(source["startedAt"], null);
	        this.endedAt = this.convertValues(source["endedAt"], null);
	        this.images = source["images"];
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
	export class ImagePayload {
	    data: string;
	    name?: string;
	    ext?: string;
	
	    static createFrom(source: any = {}) {
	        return new ImagePayload(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data = source["data"];
	        this.name = source["name"];
	        this.ext = source["ext"];
	    }
	}

}

export namespace models {
	
	export class Target {
	    id: string;
	    type: string;
	    value: string;
	    languages: string[];
	    tags: string[];
	    preferred_engines?: string[];
	
	    static createFrom(source: any = {}) {
	        return new Target(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.value = source["value"];
	        this.languages = source["languages"];
	        this.tags = source["tags"];
	        this.preferred_engines = source["preferred_engines"];
	    }
	}

}

export namespace module {
	
	export class ParamSpec {
	    name: string;
	    type: string;
	    description: string;
	    dataSource: string;
	    requireKeywords?: boolean;
	    excludedTypes?: string[];
	    allowedTypes?: string[];
	    autoStageToViewport?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ParamSpec(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.description = source["description"];
	        this.dataSource = source["dataSource"];
	        this.requireKeywords = source["requireKeywords"];
	        this.excludedTypes = source["excludedTypes"];
	        this.allowedTypes = source["allowedTypes"];
	        this.autoStageToViewport = source["autoStageToViewport"];
	    }
	}
	export class CommandDef {
	    id: string;
	    name: string;
	    trigger: string;
	    description: string;
	    params: ParamSpec[];
	    namespace?: string;
	    isPrimary?: boolean;
	    subCommand?: string;
	
	    static createFrom(source: any = {}) {
	        return new CommandDef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.trigger = source["trigger"];
	        this.description = source["description"];
	        this.params = this.convertValues(source["params"], ParamSpec);
	        this.namespace = source["namespace"];
	        this.isPrimary = source["isPrimary"];
	        this.subCommand = source["subCommand"];
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
	export class LLMConfig {
	    provider: string;
	    api_key: string;
	    base_url: string;
	    default_model: string;
	
	    static createFrom(source: any = {}) {
	        return new LLMConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.api_key = source["api_key"];
	        this.base_url = source["base_url"];
	        this.default_model = source["default_model"];
	    }
	}

}

