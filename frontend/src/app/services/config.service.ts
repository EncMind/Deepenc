import { Injectable } from '@angular/core';
import { environment } from '../../environments/environment';

export interface FirebaseClientConfig {
  apiKey: string;
  authDomain: string;
  projectId: string;
  storageBucket: string;
  messagingSenderId: string;
  appId: string;
  measurementId?: string;
}

@Injectable({
  providedIn: 'root'
})
export class ConfigService {
  private firebaseConfigCache: FirebaseClientConfig | null = null;
  private configPromise: Promise<FirebaseClientConfig> | null = null;
  private debug(...args: unknown[]): void {
    if (!environment.debugMode || !args.length) {
      return;
    }
    console.log('[ConfigService]', ...args);
  }

  constructor() {}

  async getFirebaseConfig(): Promise<FirebaseClientConfig> {
    // Return cached config if available
    if (this.firebaseConfigCache) {
      return this.firebaseConfigCache;
    }

    // Return existing promise if one is in progress
    if (this.configPromise) {
      return this.configPromise;
    }

    // Create new promise to fetch config
    this.configPromise = this.fetchFirebaseConfig();
    return this.configPromise;
  }

  private async fetchFirebaseConfig(): Promise<FirebaseClientConfig> {
    try {
      this.debug('Fetching Firebase configuration from backend...');

      const response = await fetch('/api/config/firebase', {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
        cache: 'default'
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch Firebase config: ${response.status} ${response.statusText}`);
      }

      const config: FirebaseClientConfig = await response.json();

      // Validate the configuration
      if (!this.isFirebaseConfigValid(config)) {
        throw new Error('Invalid Firebase configuration received from backend');
      }

      // Cache the configuration
      this.firebaseConfigCache = config;
      this.debug('Firebase configuration loaded successfully');

      return config;
    } catch (error) {
      console.error('Failed to fetch Firebase configuration:', error);

      this.configPromise = null;

      console.error('Backend configuration required - no fallback available');
      throw error;
    }
  }

  private isFirebaseConfigValid(config: any): config is FirebaseClientConfig {
    return config &&
           typeof config.apiKey === 'string' &&
           typeof config.authDomain === 'string' &&
           typeof config.projectId === 'string' &&
           typeof config.storageBucket === 'string' &&
           typeof config.messagingSenderId === 'string' &&
           typeof config.appId === 'string' &&
           config.apiKey.length > 0 &&
           config.authDomain.length > 0 &&
           config.projectId.length > 0 &&
           config.storageBucket.length > 0 &&
           config.messagingSenderId.length > 0 &&
           config.appId.length > 0;
  }

  async waitForFirebaseConfig(): Promise<FirebaseClientConfig> {
    return this.getFirebaseConfig();
  }

  isFirebaseConfigValidSync(config: FirebaseClientConfig | null): boolean {
    return config !== null && this.isFirebaseConfigValid(config);
  }

  clearCache(): void {
    this.firebaseConfigCache = null;
    this.configPromise = null;
  }
}
