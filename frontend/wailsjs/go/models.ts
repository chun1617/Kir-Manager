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

