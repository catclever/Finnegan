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

