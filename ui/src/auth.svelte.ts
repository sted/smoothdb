export interface User {
    aud: string;
    role: string;
    email?: string; // Optional in db mode
    username?: string; // For db mode
    // Add other fields as needed
}

export interface AuthState {
    user: User | null;
    token: string | null;
    isAuthenticated: boolean;
    isLoading: boolean;
}

class AuthStore {
    private state = $state<AuthState>({
        user: null,
        token: null,
        isAuthenticated: false,
        isLoading: true
    });

    constructor() {
        this.initializeFromStorage();
    }

    private initializeFromStorage() {
        const token = sessionStorage.getItem('smoothdb_token');
        const userStr = sessionStorage.getItem('smoothdb_user');

        if (token && userStr) {
            try {
                const user = JSON.parse(userStr);
                this.state.token = token;
                this.state.user = user;
                this.state.isAuthenticated = true;
            } catch (e) {
                console.error('[Auth] Failed to parse stored user data:', e);
                this.clearAuth();
            }
        }
        this.state.isLoading = false;
    }

    get user() {
        return this.state.user;
    }

    get token() {
        return this.state.token;
    }

    get isAuthenticated() {
        return this.state.isAuthenticated;
    }

    get isLoading() {
        return this.state.isLoading;
    }

    setAuth(token: string, user: User) {

        this.state.token = token;
        this.state.user = user;
        this.state.isAuthenticated = true;

        // Store access token and user info in sessionStorage
        sessionStorage.setItem('smoothdb_token', token);
        sessionStorage.setItem('smoothdb_user', JSON.stringify(user));
    }

    clearAuth() {
        this.state.token = null;
        this.state.user = null;
        this.state.isAuthenticated = false;

        // Clear from sessionStorage
        sessionStorage.removeItem('smoothdb_token');
        sessionStorage.removeItem('smoothdb_user');
    }

    getAuthHeaders(): Record<string, string> {
        if (this.state.token) {
            return {
                'Authorization': `Bearer ${this.state.token}`
            };
        }
        return {};
    }
}

// Simple auth store - only handles access tokens + user info
// Refresh tokens are handled by server via httpOnly cookies
export const authStore = new AuthStore();
