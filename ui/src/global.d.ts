declare global {
    interface Window {
        // written by the server into the page, see api/ui.go
        SMOOTHDB_CONFIG?: {
            loginMode: string;
            allowAnon: boolean;
            adminURL: string;
            apiURL: string;
            shortAPIURL: boolean;
        };
    }
}

export {};
