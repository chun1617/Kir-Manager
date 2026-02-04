export namespace kiroprocess {
	
	export class ProcessInfo {
	    pid: number;
	    name: string;
	    exePath: string;
	
	    static createFrom(source: any = {}) {
	        return new ProcessInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pid = source["pid"];
	        this.name = source["name"];
	        this.exePath = source["exePath"];
	    }
	}

}

export namespace main {
	
	export class AppSettings {
	    lowBalanceThreshold: number;
	    kiroVersion: string;
	    useAutoDetect: boolean;
	    customKiroInstallPath: string;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lowBalanceThreshold = source["lowBalanceThreshold"];
	        this.kiroVersion = source["kiroVersion"];
	        this.useAutoDetect = source["useAutoDetect"];
	        this.customKiroInstallPath = source["customKiroInstallPath"];
	    }
	}
	export class RefreshIntervalDTO {
	    minBalance: number;
	    maxBalance: number;
	    interval: number;
	
	    static createFrom(source: any = {}) {
	        return new RefreshIntervalDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.minBalance = source["minBalance"];
	        this.maxBalance = source["maxBalance"];
	        this.interval = source["interval"];
	    }
	}
	export class AutoSwitchSettingsDTO {
	    enabled: boolean;
	    balanceThreshold: number;
	    minTargetBalance: number;
	    folderIds: string[];
	    subscriptionTypes: string[];
	    refreshIntervals: RefreshIntervalDTO[];
	    notifyOnSwitch: boolean;
	    notifyOnLowBalance: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AutoSwitchSettingsDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.balanceThreshold = source["balanceThreshold"];
	        this.minTargetBalance = source["minTargetBalance"];
	        this.folderIds = source["folderIds"];
	        this.subscriptionTypes = source["subscriptionTypes"];
	        this.refreshIntervals = this.convertValues(source["refreshIntervals"], RefreshIntervalDTO);
	        this.notifyOnSwitch = source["notifyOnSwitch"];
	        this.notifyOnLowBalance = source["notifyOnLowBalance"];
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
	export class AutoSwitchStatus {
	    status: string;
	    lastBalance: number;
	    cooldownRemaining: number;
	    switchCount: number;
	
	    static createFrom(source: any = {}) {
	        return new AutoSwitchStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.lastBalance = source["lastBalance"];
	        this.cooldownRemaining = source["cooldownRemaining"];
	        this.switchCount = source["switchCount"];
	    }
	}
	export class BackupItem {
	    name: string;
	    backupTime: string;
	    hasToken: boolean;
	    hasMachineId: boolean;
	    machineId: string;
	    provider: string;
	    isCurrent: boolean;
	    isOriginalMachine: boolean;
	    isTokenExpired: boolean;
	    subscriptionTitle: string;
	    usageLimit: number;
	    currentUsage: number;
	    balance: number;
	    isLowBalance: boolean;
	    cachedAt: string;
	    folderId: string;
	
	    static createFrom(source: any = {}) {
	        return new BackupItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.backupTime = source["backupTime"];
	        this.hasToken = source["hasToken"];
	        this.hasMachineId = source["hasMachineId"];
	        this.machineId = source["machineId"];
	        this.provider = source["provider"];
	        this.isCurrent = source["isCurrent"];
	        this.isOriginalMachine = source["isOriginalMachine"];
	        this.isTokenExpired = source["isTokenExpired"];
	        this.subscriptionTitle = source["subscriptionTitle"];
	        this.usageLimit = source["usageLimit"];
	        this.currentUsage = source["currentUsage"];
	        this.balance = source["balance"];
	        this.isLowBalance = source["isLowBalance"];
	        this.cachedAt = source["cachedAt"];
	        this.folderId = source["folderId"];
	    }
	}
	export class CurrentUsageInfo {
	    subscriptionTitle: string;
	    usageLimit: number;
	    currentUsage: number;
	    balance: number;
	    isLowBalance: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CurrentUsageInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.subscriptionTitle = source["subscriptionTitle"];
	        this.usageLimit = source["usageLimit"];
	        this.currentUsage = source["currentUsage"];
	        this.balance = source["balance"];
	        this.isLowBalance = source["isLowBalance"];
	    }
	}
	export class FolderItem {
	    id: string;
	    name: string;
	    createdAt: string;
	    order: number;
	    snapshotCount: number;
	
	    static createFrom(source: any = {}) {
	        return new FolderItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.createdAt = source["createdAt"];
	        this.order = source["order"];
	        this.snapshotCount = source["snapshotCount"];
	    }
	}
	export class OAuthLoginResult {
	    success: boolean;
	    message: string;
	    accessToken?: string;
	    refreshToken?: string;
	    expiresAt?: string;
	    provider?: string;
	    authMethod?: string;
	    clientId?: string;
	    clientSecret?: string;
	    clientIdHash?: string;
	    userCode?: string;
	    verificationUri?: string;
	
	    static createFrom(source: any = {}) {
	        return new OAuthLoginResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.accessToken = source["accessToken"];
	        this.refreshToken = source["refreshToken"];
	        this.expiresAt = source["expiresAt"];
	        this.provider = source["provider"];
	        this.authMethod = source["authMethod"];
	        this.clientId = source["clientId"];
	        this.clientSecret = source["clientSecret"];
	        this.clientIdHash = source["clientIdHash"];
	        this.userCode = source["userCode"];
	        this.verificationUri = source["verificationUri"];
	    }
	}
	export class PathDetectionResult {
	    path: string;
	    success: boolean;
	    triedStrategies?: string[];
	    failureReasons?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new PathDetectionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.success = source["success"];
	        this.triedStrategies = source["triedStrategies"];
	        this.failureReasons = source["failureReasons"];
	    }
	}
	
	export class Result {
	    success: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	    }
	}
	export class SoftResetStatus {
	    isPatched: boolean;
	    hasCustomId: boolean;
	    customMachineId: string;
	    extensionPath: string;
	    isSupported: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SoftResetStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isPatched = source["isPatched"];
	        this.hasCustomId = source["hasCustomId"];
	        this.customMachineId = source["customMachineId"];
	        this.extensionPath = source["extensionPath"];
	        this.isSupported = source["isSupported"];
	    }
	}
	export class UsageCacheResult {
	    success: boolean;
	    message: string;
	    subscriptionTitle: string;
	    usageLimit: number;
	    currentUsage: number;
	    balance: number;
	    isLowBalance: boolean;
	    isTokenExpired: boolean;
	    cachedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new UsageCacheResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.subscriptionTitle = source["subscriptionTitle"];
	        this.usageLimit = source["usageLimit"];
	        this.currentUsage = source["currentUsage"];
	        this.balance = source["balance"];
	        this.isLowBalance = source["isLowBalance"];
	        this.isTokenExpired = source["isTokenExpired"];
	        this.cachedAt = source["cachedAt"];
	    }
	}
	export class WindowSize {
	    width: number;
	    height: number;
	
	    static createFrom(source: any = {}) {
	        return new WindowSize(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.width = source["width"];
	        this.height = source["height"];
	    }
	}

}

