import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { forkJoin } from 'rxjs';
import { take } from 'rxjs/operators';
import {
  BillingService,
  UsageStats,
  AnalyticsData,
  ProviderUsage
} from '../services/billing.service';

@Component({
  selector: 'app-usage-dashboard',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="usage-dashboard">
      <div class="dashboard-header" [class.centered]="accessRestricted()">
        <h1>Usage Dashboard</h1>
        <div class="tier-info" *ngIf="usageStats() && !accessRestricted()">
          <span class="tier-badge tier-{{ usageStats()!.currentTier }}">
            {{ getTierDisplayName(usageStats()!.currentTier) }}
          </span>
          <button class="upgrade-btn" (click)="goToPricing()">
            Upgrade Plan
          </button>
        </div>
      </div>

      <ng-container *ngIf="accessRestricted(); else usageContent">
        <div class="restricted-card">
          <h2>Usage analytics are included with paid plans</h2>
          <p>Upgrade your Deepenc plan to unlock detailed usage tracking and provider breakdowns.</p>
          <button class="upgrade-btn" (click)="goToPricing()">View Pricing</button>
        </div>
      </ng-container>

      <ng-template #usageContent>
        <!-- Loading State -->
        <div class="loading-state" *ngIf="loading()">
          <div class="spinner"></div>
          <p>Loading usage data...</p>
        </div>

        <!-- Dashboard Content -->
        <div class="dashboard-content" *ngIf="!loading() && usageStats()">
        
        <!-- Usage Overview Cards -->
        <div class="overview-cards">
          <div class="usage-card primary">
            <div class="card-header">
              <h3>Current Usage</h3>
              <div class="usage-percentage" [class.warning]="getUsageWarning() !== 'low'">
                {{ usageStats()!.usagePercentage.toFixed(1) }}%
              </div>
            </div>
            <div class="usage-bar-container">
              <div class="usage-bar">
                <div class="usage-fill" 
                     [style.width.%]="usageStats()!.usagePercentage"
                     [class.warning]="getUsageWarning() === 'medium'"
                     [class.danger]="getUsageWarning() === 'high'"
                     [class.critical]="getUsageWarning() === 'critical'">
                </div>
              </div>
              <div class="usage-text">
                {{ formatTokens(usageStats()!.tokensUsed) }} / {{ formatTokens(usageStats()!.tokensLimit) }}
              </div>
            </div>
          </div>


          <div class="usage-card">
            <div class="card-header">
              <h3>Current Tier</h3>
              <div class="stat-value tier-name">
                {{ getTierDisplayName(usageStats()!.currentTier) }}
              </div>
            </div>
            <p class="card-subtitle">
              {{ getDaysUntilReset() }} days until reset
            </p>
          </div>

          <div class="usage-card">
            <div class="card-header">
              <h3>Total Requests</h3>
              <div class="stat-value">
                {{ getTotalRequests() }}
              </div>
            </div>
            <p class="card-subtitle">
              Across all providers
            </p>
          </div>
        </div>


        <!-- Provider Details -->
        <div class="provider-details">
          <h3>Provider Breakdown</h3>
          <div class="provider-cards">
            <div class="provider-card" *ngFor="let provider of getProviderStats(); trackBy: trackByProvider">
              <div class="provider-header">
                <div class="provider-name">
                  <div class="provider-icon" [class]="provider.name.toLowerCase()"></div>
                  {{ getProviderDisplayName(provider.name) }}
                </div>
                <div class="provider-rate">
                  Rate: {{ provider.usage.conversionRate }}x
                </div>
              </div>
              
              <div class="provider-stats">
                <div class="stat-row">
                  <span class="stat-label">Actual Tokens:</span>
                  <span class="stat-value">{{ formatTokens(provider.usage.actualTokensUsed) }}</span>
                </div>
                <div class="stat-row">
                  <span class="stat-label">Equivalent Tokens:</span>
                  <span class="stat-value">{{ formatTokens(provider.usage.equivalentTokens) }}</span>
                </div>
                <div class="stat-row">
                  <span class="stat-label">Requests:</span>
                  <span class="stat-value">{{ provider.usage.requestCount }}</span>
                </div>
              </div>

              <div class="provider-bar">
                <div class="provider-fill" 
                     [style.width.%]="getProviderPercentage(provider.usage.equivalentTokens)">
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Conversion Rates Info -->
        <div class="conversion-info" *ngIf="analyticsData()?.conversionRates">
          <h3>Token Conversion Rates</h3>
          <p class="conversion-explanation">
            Different AI providers have different costs. We normalize usage using conversion rates:
          </p>
          <div class="conversion-rates">
            <div class="rate-item" *ngFor="let rate of getConversionRates()">
              <span class="provider-name">{{ getProviderDisplayName(rate.provider) }}</span>
              <span class="rate-value">{{ rate.rate }}x GPT equivalent</span>
            </div>
          </div>
        </div>

        </div>

        <!-- Error State -->
        <div class="error-state" *ngIf="error()">
          <div class="error-icon">⚠️</div>
          <h3>Unable to Load Usage Data</h3>
          <p>{{ error() }}</p>
          <button class="retry-btn" (click)="loadData()">Retry</button>
        </div>
      </ng-template>
    </div>
  `,
  styleUrls: ['./usage-dashboard.component.css']
})
export class UsageDashboardComponent implements OnInit {
  private readonly billingService = inject(BillingService);
  private readonly router = inject(Router);

  private readonly TIER_NAMES: Record<string, string> = {
    'free': 'Free',
    'plus': 'Plus',
    'pro': 'Pro',
    'pro_plus': 'Pro Plus',
    'free_trial': 'Free Trial'
  };

  private readonly PROVIDER_NAMES: Record<string, string> = {
    'openai': 'OpenAI',
    'anthropic': 'Claude',
    'gemini': 'Gemini'
  };

  // State signals
  loading = signal(true);
  error = signal<string>('');
  usageStats = signal<UsageStats | null>(null);
  analyticsData = signal<AnalyticsData | null>(null);
  accessRestricted = signal(false);

  // Cached computed values to prevent infinite change detection
  providerStats = signal<{ name: string, usage: ProviderUsage }[]>([]);
  conversionRates = signal<{ provider: string, rate: number }[]>([]);
  totalRequests = signal<number>(0);

  ngOnInit() {
    // Check if user is on free tier and redirect to account page
    this.billingService.getSubscriptionInfo().pipe(take(1)).subscribe({
      next: (info) => {
        if (!info || info.tier === 'free') {
          this.accessRestricted.set(true);
          this.loading.set(false);
          this.error.set('');
          this.usageStats.set(null);
          this.analyticsData.set(null);
          return;
        }
        this.accessRestricted.set(false);
        this.loadData();
      },
      error: () => {
        this.accessRestricted.set(true);
        this.loading.set(false);
        this.error.set('');
        this.usageStats.set(null);
        this.analyticsData.set(null);
      }
    });
  }


  loadData() {
    this.accessRestricted.set(false);
    this.loading.set(true);
    this.error.set('');

    forkJoin({
      stats: this.billingService.getUsageStats(),
      analytics: this.billingService.getUsageAnalytics()
    }).subscribe({
      next: ({ stats, analytics }) => {
        this.usageStats.set(stats);
        this.analyticsData.set(analytics);
        this.updateComputedValues(stats, analytics);
        this.loading.set(false);
      },
      error: (error) => {
        console.error('[UsageDashboard] Error loading data:', error);
        this.error.set('Failed to load usage data. Please try again.');
        this.loading.set(false);
      }
    });
  }


  private updateComputedValues(stats: UsageStats, analytics: AnalyticsData) {
    if (stats?.providerUsage) {
      const providers = Object.entries(stats.providerUsage)
        .map(([name, usage]) => ({ name, usage }))
        .filter(p => p.usage.equivalentTokens > 0);
      this.providerStats.set(providers);

      const total = Object.values(stats.providerUsage)
        .reduce((sum, usage) => sum + usage.requestCount, 0);
      this.totalRequests.set(total);
    }

    if (analytics?.conversionRates) {
      const rates = Object.entries(analytics.conversionRates)
        .map(([provider, rate]) => ({ provider, rate }));
      this.conversionRates.set(rates);
    }
  }

  getUsageWarning(): 'low' | 'medium' | 'high' | 'critical' {
    const stats = this.usageStats();
    if (!stats) return 'low';
    return this.billingService.getUsageWarningLevel(stats.usagePercentage);
  }

  getTierDisplayName(tier: string): string {
    return this.TIER_NAMES[tier] || tier;
  }

  getProviderDisplayName(provider: string): string {
    return this.PROVIDER_NAMES[provider] || provider;
  }

  formatTokens(tokens: number): string {
    return this.billingService.formatTokens(tokens);
  }

  getDaysUntilReset(): number {
    const stats = this.usageStats();
    if (!stats) return 0;
    return this.billingService.getDaysUntilReset(stats.billingPeriodEnd);
  }

  getTotalRequests(): number {
    return this.totalRequests();
  }

  getProviderStats(): { name: string, usage: ProviderUsage }[] {
    return this.providerStats();
  }

  getProviderPercentage(equivalentTokens: number): number {
    const stats = this.usageStats();
    if (!stats?.tokensUsed || stats.tokensUsed === 0) return 0;
    return (equivalentTokens / stats.tokensUsed) * 100;
  }

  getConversionRates(): { provider: string, rate: number }[] {
    return this.conversionRates();
  }

  goToPricing() {
    this.router.navigate(['/pricing']);
  }

  trackByProvider(index: number, item: { name: string, usage: ProviderUsage }): string {
    return item.name;
  }
}
