import { Injectable, inject } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Observable, BehaviorSubject, map, tap, of, Subscription, distinctUntilChanged, filter } from 'rxjs';
import { AuthService } from '../auth.service';
import { environment } from '../../environments/environment';

export interface TierInfo {
  id: string;
  name: string;
  price: number; // Price in cents
  tokenLimit: number;
  allowedModels: { [provider: string]: string[] };
  features: string[];
}

export interface UsageStats {
  tokensUsed: number;
  tokensLimit: number;
  remainingTokens: number;
  usagePercentage: number;
  currentTier: string;
  freeTrial?: boolean;
  billingPeriodEnd: number;
  providerUsage: { [provider: string]: ProviderUsage };
  allowedModels: { [provider: string]: string[] };
}

export interface ProviderUsage {
  actualTokensUsed: number;
  equivalentTokens: number;
  conversionRate: number;
  requestCount: number;
}

export interface SubscriptionInfo {
  tier: string;
  status: string;
  currentPeriodStart: number;
  currentPeriodEnd: number;
  usage: {
    tokensUsed: number;
    tokensLimit: number;
    usagePercentage: number;
    providerBreakdown: { [provider: string]: ProviderUsage };
    freeTrial?: boolean;
  };
  freeTrial?: {
    active: boolean;
    startAt?: number;
    endAt?: number;
  };
}

export interface AnalyticsData {
  currentPeriod: {
    start: number;
    end: number;
    tokensUsed: number;
    tokensLimit: number;
  };
  dailyUsage: DailyUsage[];
  providerUsage: { [provider: string]: ProviderUsage };
  tier: string;
  subscriptionStatus: string;
  conversionRates: { [provider: string]: number };
}

export interface DailyUsage {
  date: string;
  tokensUsed: number;
  providerBreakdown: { [provider: string]: ProviderUsage };
}

export interface CheckoutResponse {
  sessionId: string;
  url: string;
}

export interface SubscriptionStatus {
  tier: string;
  status: string;
  currentPeriodEnd: number;
  nextBillingDate: number;
  cancelAtPeriodEnd: boolean;
  hasPaymentMethod: boolean;
  pastDue: boolean;
  freeTrialActive?: boolean;
  freeTrialStartAt?: number;
  freeTrialEndAt?: number;
}

@Injectable({
  providedIn: 'root'
})
export class BillingService {
  private http = inject(HttpClient);
  private auth = inject(AuthService);
  
  private usageStatsSubject = new BehaviorSubject<UsageStats | null>(null);
  public usageStats$ = this.usageStatsSubject.asObservable();
  
  private tiersSubject = new BehaviorSubject<TierInfo[]>([]);
  public tiers$ = this.tiersSubject.asObservable();

  private subscriptionInfoSubject = new BehaviorSubject<SubscriptionInfo | null>(null);
  public subscriptionInfo$ = this.subscriptionInfoSubject.asObservable();
  private subscriptionRequestSeq = 0;
  private activeSubscriptionUserId: string | null = null;
  private usageRequestSeq = 0;
  private activeUsageUserId: string | null = null;

  private baseUrl = ''; // Use relative URLs for all environments
  private authSubscription: Subscription | null = null;

  constructor() {
    this.subscribeToAuthChanges();
    this.loadTiers();
  }

  private subscribeToAuthChanges(): void {
    this.authSubscription = this.auth.user$
      .pipe(distinctUntilChanged((prev, curr) => prev?.uid === curr?.uid))
      .subscribe((user) => {
        const userId = user?.uid || null;
        const changed = userId !== this.activeSubscriptionUserId;

        if (changed) {
          this.clearCachedSubscriptionInfo();
          this.clearCachedUsageStats();
        }

        if (userId) {
          this.activeSubscriptionUserId = userId;
          this.activeUsageUserId = userId;
          this.refreshSubscriptionInfo();
          this.refreshUsageStats();
        }
      });
  }

  private async getAuthHeaders(): Promise<HttpHeaders> {
    const token = await this.auth.getIdToken();
    return new HttpHeaders({
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    });
  }

  private makeAuthenticatedRequest<T>(method: string, endpoint: string, body?: any): Observable<T> {
    return new Observable(observer => {
      this.getAuthHeaders().then(headers => {
        const url = `${this.baseUrl}${endpoint}`;
        let request: Observable<T>;
        
        switch (method.toUpperCase()) {
          case 'GET':
            request = this.http.get<T>(url, { headers });
            break;
          case 'POST':
            request = this.http.post<T>(url, body, { headers });
            break;
          case 'PUT':
            request = this.http.put<T>(url, body, { headers });
            break;
          case 'DELETE':
            request = this.http.delete<T>(url, { headers });
            break;
          default:
            observer.error(new Error(`Unsupported method: ${method}`));
            return;
        }
        
        request.subscribe({
          next: data => observer.next(data),
          error: error => observer.error(error),
          complete: () => observer.complete()
        });
      }).catch(error => observer.error(error));
    });
  }

  // Public tier information (no auth required)
  getTiers(): Observable<{ tiers: TierInfo[] }> {
    return this.http.get<{ tiers: TierInfo[] }>(`${this.baseUrl}/api/usage/tiers`);
  }

  loadTiers(): void {
    this.getTiers().subscribe({
      next: (response) => {
        this.tiersSubject.next(response.tiers);
      },
      error: (error) => {
        console.error('Failed to load tiers:', error);
      }
    });
  }

  // Usage statistics (auth required)
  getUsageStats(): Observable<UsageStats> {
    const currentUser = this.auth.getCurrentUser();
    const userId = currentUser?.uid || null;

    if (!userId) {
      this.clearCachedUsageStats();
      const emptyStats: UsageStats = {
        tokensUsed: 0,
        tokensLimit: 0,
        remainingTokens: 0,
        usagePercentage: 0,
        currentTier: 'free',
        billingPeriodEnd: 0,
        providerUsage: {},
        allowedModels: {}
      };
      return of(emptyStats);
    }

    const requestId = ++this.usageRequestSeq;
    this.activeUsageUserId = userId;

    return this.makeAuthenticatedRequest<UsageStats>('GET', `/api/usage/stats`).pipe(
      map((stats) => {
        const latestUserId = this.auth.getCurrentUser()?.uid || null;
        const isCurrentUser = latestUserId === userId && requestId === this.usageRequestSeq;

        if (!isCurrentUser) {
          console.warn('[BillingService] Ignoring stale usage stats response for user', userId);
          return null as unknown as UsageStats;
        }

        return stats;
      }),
      filter((stats): stats is UsageStats => !!stats),
      tap((stats) => this.usageStatsSubject.next(stats))
    );
  }

  refreshUsageStats(): void {
    this.getUsageStats().subscribe({
      error: (error) => {
        console.error('Failed to load usage stats:', error);
      }
    });
  }

  // Detailed analytics (auth required)
  getUsageAnalytics(): Observable<AnalyticsData> {
    return this.makeAuthenticatedRequest<AnalyticsData>('GET', `/api/usage/analytics`);
  }

  // Subscription information (auth required)
  private fetchSubscriptionInfo(): Observable<SubscriptionInfo> {
    const currentUser = this.auth.getCurrentUser();
    const userId = currentUser?.uid || null;

    if (!userId) {
      // No authenticated user; ensure cache reflects anonymous state.
      this.subscriptionInfoSubject.next(null);
      return of({
        tier: 'free',
        status: 'inactive',
        currentPeriodStart: 0,
        currentPeriodEnd: 0,
        usage: {
          tokensUsed: 0,
          tokensLimit: 0,
          usagePercentage: 0,
          providerBreakdown: {} as { [provider: string]: ProviderUsage }
        }
      } as SubscriptionInfo);
    }

    const requestId = ++this.subscriptionRequestSeq;
    this.activeSubscriptionUserId = userId;

    return this.makeAuthenticatedRequest<SubscriptionInfo>('GET', `/api/subscription`).pipe(
      map(response => response || { tier: 'free' } as SubscriptionInfo),
      tap(info => {
        if (this.activeSubscriptionUserId !== userId || requestId !== this.subscriptionRequestSeq) {
          console.warn('[BillingService] Ignoring stale subscription info for user', userId);
          return;
        }
        this.subscriptionInfoSubject.next(info);
      })
    );
  }

  getSubscriptionInfo(): Observable<SubscriptionInfo> {
    return this.fetchSubscriptionInfo();
  }

  refreshSubscriptionInfo(): void {
    this.fetchSubscriptionInfo().subscribe({
      error: (error) => {
        console.error('Failed to refresh subscription info:', error);
      }
    });
  }

  clearCachedSubscriptionInfo(): void {
    this.subscriptionInfoSubject.next(null);
    this.activeSubscriptionUserId = null;
    this.subscriptionRequestSeq++;
  }

  getCachedSubscriptionInfo(): SubscriptionInfo | null {
    return this.subscriptionInfoSubject.value;
  }

  getSubscriptionStatus(): Observable<SubscriptionStatus> {
    return this.makeAuthenticatedRequest<SubscriptionStatus>('GET', `/api/subscription/status`);
  }

  // Stripe integration
  createCheckoutSession(tier: string): Observable<CheckoutResponse> {
    return this.makeAuthenticatedRequest<CheckoutResponse>('POST', `/api/subscription/checkout`, { tier });
  }

  getBillingPortalUrl(): Observable<{ url: string }> {
    return this.makeAuthenticatedRequest<{ url: string }>('POST', `/api/subscription/portal`, {});
  }

  cancelSubscription(): Observable<{ success: boolean; message: string }> {
    return this.makeAuthenticatedRequest<{ success: boolean; message: string }>('POST', `/api/subscription/cancel`, {});
  }

  renewSubscription(): Observable<any> {
    return this.makeAuthenticatedRequest<any>('POST', `/api/subscription/renew`, {});
  }

  // Helper methods
  isUpgradeRequired(model: string, currentTier: string): boolean {
    const tiers = this.tiersSubject.value;
    const tier = tiers.find(t => t.id === currentTier);
    if (!tier) return true;

    // Check if model is available in current tier
    for (const [provider, models] of Object.entries(tier.allowedModels)) {
      if (models.includes(model)) {
        return false;
      }
    }
    return true;
  }

  getModelProvider(model: string): string {
    if (model.startsWith('gpt')) return 'openai';
    if (model.startsWith('claude')) return 'anthropic';
    if (model.startsWith('gemini')) return 'gemini';
    return 'unknown';
  }

  getUsageWarningLevel(usagePercentage: number): 'low' | 'medium' | 'high' | 'critical' {
    if (usagePercentage >= 95) return 'critical';
    if (usagePercentage >= 90) return 'high';
    if (usagePercentage >= 80) return 'medium';
    return 'low';
  }

  canMakeRequest(estimatedTokens: number): boolean {
    const stats = this.usageStatsSubject.value;
    if (!stats) return true;
    
    return stats.remainingTokens >= estimatedTokens;
  }

  formatTokens(tokens: number): string {
    if (tokens >= 1000000) {
      return `${(tokens / 1000000).toFixed(1)}M`;
    }
    if (tokens >= 1000) {
      return `${(tokens / 1000).toFixed(1)}K`;
    }
    return tokens.toString();
  }

  getDaysUntilReset(periodEnd: number): number {
    const now = Date.now();
    const msUntilReset = periodEnd - now;
    return Math.ceil(msUntilReset / (1000 * 60 * 60 * 24));
  }
  
  getCachedUsageStats(): UsageStats | null {
    return this.usageStatsSubject.value;
  }

  clearCachedUsageStats(): void {
    this.usageStatsSubject.next(null);
    this.activeUsageUserId = null;
    this.usageRequestSeq++;
  }
}
