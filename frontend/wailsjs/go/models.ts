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

export namespace main {
	
	export class Card {
	    id: string;
	    type: string;
	    title: string;
	    content: string;
	    sourceUrl?: string;
	    originalText?: string;
	    translatedText?: string;
	    publishedAt?: string;
	    authorName?: string;
	    authorAvatar?: string;
	    echoOf?: string;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Card(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.title = source["title"];
	        this.content = source["content"];
	        this.sourceUrl = source["sourceUrl"];
	        this.originalText = source["originalText"];
	        this.translatedText = source["translatedText"];
	        this.publishedAt = source["publishedAt"];
	        this.authorName = source["authorName"];
	        this.authorAvatar = source["authorAvatar"];
	        this.echoOf = source["echoOf"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
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
	
	    static createFrom(source: any = {}) {
	        return new ParamSpec(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.description = source["description"];
	        this.dataSource = source["dataSource"];
	    }
	}
	export class CommandDef {
	    id: string;
	    name: string;
	    trigger: string;
	    description: string;
	    params: ParamSpec[];
	
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

