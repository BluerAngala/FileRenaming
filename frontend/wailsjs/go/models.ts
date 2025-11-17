export namespace main {
	
	export class AIConfig {
	    apiKey: string;
	    baseURL: string;
	    model: string;
	
	    static createFrom(source: any = {}) {
	        return new AIConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.apiKey = source["apiKey"];
	        this.baseURL = source["baseURL"];
	        this.model = source["model"];
	    }
	}
	export class FileInfo {
	    path: string;
	    name: string;
	    isDir: boolean;
	    fullPath: string;
	
	    static createFrom(source: any = {}) {
	        return new FileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.isDir = source["isDir"];
	        this.fullPath = source["fullPath"];
	    }
	}
	export class ModelInfo {
	    id: string;
	    object: string;
	    created: number;
	    owned_by: string;
	
	    static createFrom(source: any = {}) {
	        return new ModelInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.object = source["object"];
	        this.created = source["created"];
	        this.owned_by = source["owned_by"];
	    }
	}
	export class PromptTemplate {
	    name: string;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new PromptTemplate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.content = source["content"];
	    }
	}
	export class PromptTemplates {
	    templates: PromptTemplate[];
	
	    static createFrom(source: any = {}) {
	        return new PromptTemplates(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.templates = this.convertValues(source["templates"], PromptTemplate);
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
	export class RenameRule {
	    mode: string;
	    pattern: string;
	    replaceFrom: string;
	    replaceTo: string;
	    addPrefix: string;
	    addSuffix: string;
	    caseType: string;
	    numberStart: number;
	    numberStep: number;
	    aiPrompt: string;
	    aiGenerated: string[];
	
	    static createFrom(source: any = {}) {
	        return new RenameRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.pattern = source["pattern"];
	        this.replaceFrom = source["replaceFrom"];
	        this.replaceTo = source["replaceTo"];
	        this.addPrefix = source["addPrefix"];
	        this.addSuffix = source["addSuffix"];
	        this.caseType = source["caseType"];
	        this.numberStart = source["numberStart"];
	        this.numberStep = source["numberStep"];
	        this.aiPrompt = source["aiPrompt"];
	        this.aiGenerated = source["aiGenerated"];
	    }
	}

}

