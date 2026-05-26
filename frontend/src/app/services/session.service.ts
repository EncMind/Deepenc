import { Injectable } from '@angular/core';
import { environment } from '../../environments/environment';

// Interface matching the FreeSession structure exactly
export interface FreeSession {
  messageCount: number;
  startTime: number;
  dailyUsage: { date: string; count: number };
  leftSessionId: string | null;
  centerSessionId: string | null;
  rightSessionId: string | null;
  anonymousUserId: string;
  lastUpdated?: number;
  dailyUsageTimestamp?: number;
}

@Injectable({ providedIn: 'root' })
export class SessionService {
  private base = environment.apiUrl || '/api';
  private cache: { [key: string]: FreeSession } = {};
  private hasStorage(): boolean {
    return typeof window !== 'undefined' && typeof window.localStorage !== 'undefined';
  }
  private storageKey(id: string): string {
    return `deepenc_session_${id}`;
  }
  private cookieKey(id: string): string {
    return `deepenc_session_${id}`;
  }
  private readCookie(key: string): string | null {
    if (typeof document === 'undefined') return null;
    const match = document.cookie.match(new RegExp('(?:^|; )' + key.replace(/([.$?*|{}()[\\\]\/+^])/g, '\\$1') + '=([^;]*)'));
    return match ? decodeURIComponent(match[1]) : null;
  }
  private writeCookie(key: string, value: string) {
    if (typeof document === 'undefined') return;
    const expires = new Date(Date.now() + 365 * 24 * 60 * 60 * 1000).toUTCString();
    const secure = typeof window !== 'undefined' && window.location?.protocol === 'https:' ? '; Secure' : '';
    document.cookie = `${key}=${encodeURIComponent(value)}; path=/; expires=${expires}; SameSite=Lax${secure}`;
  }
  private normalizeDate(dateStr?: string | null): string | undefined {
    if (!dateStr) return undefined;
    if (/^\d{4}-\d{2}-\d{2}$/.test(dateStr)) {
      return dateStr;
    }
    const parsed = new Date(dateStr);
    if (!Number.isNaN(parsed.getTime())) {
      return parsed.toISOString().split('T')[0];
    }
    return undefined;
  }


  private normalizeSession(session: FreeSession): FreeSession {
    if (!session) {
      return {
        messageCount: 0,
        startTime: Date.now(),
        dailyUsage: { date: new Date().toISOString().split('T')[0], count: 0 },
        leftSessionId: null,
        centerSessionId: null,
        rightSessionId: null,
        anonymousUserId: '',
        lastUpdated: Date.now(),
        dailyUsageTimestamp: Date.now()
      };
    }
    session.messageCount = Math.max(0, Number(session.messageCount) || 0);
    const normalizedDate = this.normalizeDate(session.dailyUsage?.date) ?? new Date().toISOString().split('T')[0];
    const normalizedCount = Math.max(0, Number(session.dailyUsage?.count) || 0);
    session.dailyUsage = { date: normalizedDate, count: normalizedCount };
    session.dailyUsageTimestamp = Number(session.dailyUsageTimestamp) || Number(session.lastUpdated) || Date.now();
    session.lastUpdated = Number(session.lastUpdated) || Date.now();
    if (normalizedCount > session.messageCount) {
      session.messageCount = normalizedCount;
    }
    return session;
  }

  private getSessionTimestamp(session: FreeSession): number {
    return Number(session.dailyUsageTimestamp) || Number(session.lastUpdated) || 0;
  }

  private loadFromStorage(id: string): FreeSession | null {
    let raw: string | null = null;
    if (this.hasStorage()) {
      raw = localStorage.getItem(this.storageKey(id));
    }
    if (!raw) {
      raw = this.readCookie(this.cookieKey(id));
    }
    if (!raw) return null;
    try {
      const parsed = JSON.parse(raw) as FreeSession;
      // Ensure object has required shape
      if (typeof parsed?.messageCount === 'number' && parsed?.anonymousUserId === id) {
        this.normalizeSession(parsed);
        this.debug('Loaded session from localStorage fallback');
        return parsed;
      }
    } catch (err) {
      console.error('❌ SessionService: Failed to parse session from localStorage:', err);
    }
    return null;
  }
  private saveToStorage(session: FreeSession) {
    const payload = JSON.stringify(session);
    if (this.hasStorage()) {
      try {
        localStorage.setItem(this.storageKey(session.anonymousUserId), payload);
        this.debug('Saved session to localStorage');
      } catch (err) {
        console.error('❌ SessionService: Failed to persist session to localStorage:', err);
      }
    }
    this.writeCookie(this.cookieKey(session.anonymousUserId), payload);
  }
  private debug(...args: unknown[]): void {
    if (!environment.debugMode || !args.length) {
      return;
    }
    console.log('🔍 SessionService:', ...args);
  }

  async getSession(anonymousUserId: string): Promise<FreeSession> {
    this.debug('Getting session for user:', anonymousUserId);

    // Check cache first
    const cached = this.cache[anonymousUserId];
    if (cached) {
      this.debug('Using cached session');
      return cached;
    }

    // Prefer localStorage if available to avoid unnecessary network calls
    const storedFirst = this.loadFromStorage(anonymousUserId);
    if (storedFirst) {
      this.debug('Using stored session');
      this.cache[anonymousUserId] = storedFirst;
      return storedFirst;
    }

    try {
      const response = await fetch(`${this.base}/anonymous-session?anonymousUserId=${encodeURIComponent(anonymousUserId)}`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        }
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${await response.text()}`);
      }

      const sessionData = this.normalizeSession(await response.json() as FreeSession);
      this.debug('Got session from backend:', sessionData);

      // Merge with any local fallback if it has newer usage data
      const stored = this.loadFromStorage(anonymousUserId);
      if (stored && stored.dailyUsage?.date === sessionData.dailyUsage?.date) {
        const storedTs = this.getSessionTimestamp(stored);
        const serverTs = this.getSessionTimestamp(sessionData);
        if (storedTs > serverTs || (storedTs === serverTs && stored.messageCount > sessionData.messageCount)) {
          this.debug('Using local session (newer timestamp) instead of backend');
          Object.assign(sessionData, stored);
        }
      }

      // Cache & persist the authoritative session
      this.cache[anonymousUserId] = sessionData;
      this.saveToStorage(sessionData);
      return sessionData;

    } catch (error) {
      console.error('❌ SessionService: Error getting session:', error);

      // Try stored session (if not already used) before creating a new one
      const stored = this.loadFromStorage(anonymousUserId);
      if (stored) {
        this.cache[anonymousUserId] = stored;
        return stored;
      }

      // Fallback to default session if backend fails
      const defaultSession: FreeSession = this.normalizeSession({
        messageCount: 0,
        startTime: Date.now(),
        dailyUsage: { date: new Date().toISOString().split('T')[0], count: 0 },
        leftSessionId: null,
        centerSessionId: null,
        rightSessionId: null,
        anonymousUserId: anonymousUserId
      });

      this.debug('Using fallback default session');
      this.cache[anonymousUserId] = defaultSession;
      this.saveToStorage(defaultSession);
      return defaultSession;
    }
  }

  async saveSession(session: FreeSession): Promise<void> {
    const now = Date.now();
    session.lastUpdated = now;
    session.dailyUsageTimestamp = now;
    session = this.normalizeSession(session);
    this.debug('Saving session for user:', session.anonymousUserId);

    try {
      const response = await fetch(`${this.base}/anonymous-session?anonymousUserId=${encodeURIComponent(session.anonymousUserId)}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(session)
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${await response.text()}`);
      }

      this.debug('Session saved successfully');

      // Update cache
      this.cache[session.anonymousUserId] = session;
      this.saveToStorage(session);

    } catch (error) {
      console.error('❌ SessionService: Error saving session:', error);
      
      // Still update cache even if backend save fails
      this.cache[session.anonymousUserId] = session;
      this.saveToStorage(session);
    }
  }

}
