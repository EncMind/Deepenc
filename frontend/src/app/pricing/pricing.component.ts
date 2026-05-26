import { Component, DestroyRef, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { BillingService, TierInfo, UsageStats } from '../services/billing.service';
import { AuthService } from '../auth.service';
import { loadStripe, Stripe } from '@stripe/stripe-js';
import { environment } from '../../environments/environment';
import { Observable } from 'rxjs';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';

@Component({
  selector: 'app-pricing',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="pricing-page">
      <section class="plans-section">
        <div class="container">
          <div class="section-heading">
            <h2>Flexible plans for privacy-focused professionals</h2>
          </div>
          <div class="plan-grid" *ngIf="tiers().length > 0">
            <div
              class="plan-card"
              *ngFor="let tier of tiers(); trackBy: trackByTier"
              [class.plan-current]="tier.id === currentTier()"
              [class.plan-popular]="tier.id === 'pro'"
            >
              <div class="plan-badge" *ngIf="tier.id === 'pro'">
                <span>Most Popular</span>
              </div>
              <div class="plan-badge current" *ngIf="tier.id === currentTier()">
                <span>Current Plan</span>
              </div>

              <div class="plan-header">
                <h3>{{ getTierDisplayName(tier.id) }}</h3>
                <div class="plan-price">
                  <span class="price" *ngIf="tier.price === 0">Free</span>
                  <span class="price" *ngIf="tier.price > 0">
                    <span class="currency">$</span>{{ (tier.price / 100) }}<span class="period">/month</span>
                  </span>
                </div>
                <div class="plan-trial" *ngIf="tier.price > 0">
                  Monthly billing — cancel anytime
                </div>
              </div>

              <div class="plan-quota">
                <span class="quota-value">
                  {{ formatTokens(tier.tokenLimit) }} tokens / month
                </span>
                <span class="quota-note" *ngIf="tier.id !== 'free'">GPT-equivalent usage</span>
              </div>

              <ul class="plan-features">
                <li class="feature-item" *ngFor="let feature of tier.features">
                  <span class="check-icon">✓</span>
                  <span>{{ feature }}</span>
                </li>
              </ul>

              <div class="plan-models" *ngIf="tier.allowedModels">
                <h4>Model access</h4>
                <div class="model-badges">
                  <span
                    class="model-badge openai"
                    *ngIf="tier.allowedModels.openai && tier.allowedModels.openai.length > 0"
                  >
                    OpenAI: {{ formatModelList(tier.allowedModels.openai) }}
                  </span>
                  <span
                    class="model-badge anthropic"
                    *ngIf="tier.allowedModels.anthropic && tier.allowedModels.anthropic.length > 0"
                  >
                    Claude: {{ formatModelList(tier.allowedModels.anthropic) }}
                  </span>
                  <span
                    class="model-badge gemini"
                    *ngIf="tier.allowedModels.gemini && tier.allowedModels.gemini.length > 0"
                  >
                    Gemini: {{ formatModelList(tier.allowedModels.gemini) }}
                  </span>
                </div>
              </div>

              <div class="plan-action">
                <button
                  class="btn btn-primary upgrade-btn"
                  [class.current]="tier.id === currentTier() && !renewing()"
                  [disabled]="(tier.id === currentTier() && !renewing()) || upgrading() === tier.id || renewing()"
                  (click)="upgrade(tier.id)"
                >
                  <span class="spinner" *ngIf="upgrading() === tier.id"></span>
                  <span *ngIf="upgrading() === tier.id">Upgrading...</span>
                  <span *ngIf="upgrading() !== tier.id && tier.id !== currentTier() && tier.price === 0">Get Started</span>
                  <span *ngIf="upgrading() !== tier.id && tier.id !== currentTier() && tier.price > 0">
                    {{ tier.price > getCurrentTierPrice() ? 'Upgrade' : 'Change Plan' }}
                  </span>
                  <span *ngIf="tier.id === currentTier() && !renewing()">Current Plan</span>
                </button>

                <button
                  class="btn btn-outline renew-btn"
                  *ngIf="tier.id === currentTier() && tier.price > 0"
                  [disabled]="renewing()"
                  (click)="renewCurrentPlan()"
                >
                  <span *ngIf="!renewing()">🔄 Renew now &amp; reset quota</span>
                  <span *ngIf="renewing()">Renewing...</span>
                </button>

                <div class="current-plan-label" *ngIf="tier.id === currentTier()">
                  <span>✓ Current Plan</span>
                </div>
              </div>
            </div>
          </div>

          <div class="empty-state" *ngIf="!loading() && tiers().length === 0 && !error()">
            <h3>Pricing is on the way</h3>
            <p>We're fetching the latest information. Please refresh in a moment.</p>
            <button class="btn btn-primary" (click)="loadTiers()">Reload</button>
          </div>
        </div>
      </section>

      <section class="faq-section">
        <div class="container">
          <div class="section-heading">
            <h2>Frequently asked questions</h2>
          </div>
          <div class="faq-grid">
            <article class="faq-card">
              <h4>What are GPT-equivalent tokens?</h4>
              <p>We normalize spend across providers. Higher-cost models draw more tokens, while efficient models consume fewer.</p>
            </article>
            <article class="faq-card">
              <h4>Can I change plans anytime?</h4>
              <p>Absolutely. Upgrade, downgrade, or pause whenever you need. Billing prorates automatically.</p>
            </article>
            <article class="faq-card">
              <h4>What happens if I hit my quota?</h4>
              <p>Requests pause until the next cycle. Renew early or move to a higher tier to restore access immediately.</p>
            </article>
            <article class="faq-card">
              <h4>Can I try Deepenc before purchasing?</h4>
              <p>We offer 30-day evaluations for qualifying freelancers and independents. Contact support to discuss your use case and unlock trial access.</p>
            </article>
          </div>
        </div>
      </section>

      <section class="final-cta">
        <div class="container">
          <div class="cta-card">
            <h2>Ready to experience hardware-backed private AI?</h2>
            <p>Launch Deepenc today and keep every conversation secure while delivering your best work.</p>
            <div class="cta-actions">
              <a routerLink="/auth" class="cta-button">Sign up today</a>
            </div>
          </div>
        </div>
      </section>

      <div class="loading-state" *ngIf="loading()">
        <div class="spinner spinner-large"></div>
        <p>Loading pricing information...</p>
      </div>

      <div class="error-state" *ngIf="error()">
        <div class="error-icon">⚠️</div>
        <h3>Unable to load pricing</h3>
        <p>{{ error() }}</p>
        <button class="btn btn-primary" (click)="loadTiers()">Retry</button>
      </div>
    </div>
  `,
  styleUrls: ['./pricing.component.css']
})
export class PricingComponent implements OnInit {
  private billingService = inject(BillingService);
  private authService = inject(AuthService);
  private router = inject(Router);
  private destroyRef = inject(DestroyRef);

  tiers = signal<TierInfo[]>([]);
  usageStats = signal<UsageStats | null>(null);
  currentTier = signal<string>('free');
  loading = signal(true);
  error = signal<string>('');
  upgrading = signal<string>('');
  renewing = signal<boolean>(false);
  isAuthenticated = false;

  private stripe: Stripe | null = null;

  ngOnInit() {
    this.initializeStripe();
    this.billingService.usageStats$
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe((stats) => {
        this.usageStats.set(stats);
      });
    this.loadTiers();
    this.checkAuthentication();
  }

  private async initializeStripe() {
    try {
      const response = await fetch(environment.configEndpoints?.stripe || '/api/config/stripe');
      const config = await response.json();

      if (config?.publishableKey) {
        this.stripe = await loadStripe(config.publishableKey);
      } else {
        console.warn('Stripe publishable key not provided');
      }
    } catch (error) {
      console.error('Failed to initialize Stripe:', error);
    }
  }

  private async checkAuthentication() {
    try {
      const authResult: any = this.authService.isAuthenticated();

      if (authResult && typeof authResult.subscribe === 'function') {
        (authResult as Observable<boolean>).subscribe((authenticated) => {
          this.isAuthenticated = authenticated;
          if (authenticated) {
            this.loadCurrentTier();
            this.loadUsageStats();
          }
        });
        return;
      }

      if (authResult instanceof Promise) {
        const authenticated = await authResult;
        this.isAuthenticated = !!authenticated;
        if (this.isAuthenticated) {
          this.loadCurrentTier();
          this.loadUsageStats();
        }
        return;
      }

      this.isAuthenticated = !!authResult;
      if (this.isAuthenticated) {
        this.loadCurrentTier();
        this.loadUsageStats();
      }
    } catch (error) {
      console.error('Authentication check failed:', error);
      this.isAuthenticated = false;
    }
  }

  loadTiers() {
    this.loading.set(true);
    this.error.set('');

    this.billingService.getTiers().subscribe({
      next: (response) => {
        this.tiers.set(response.tiers);
        this.loading.set(false);
      },
      error: (error) => {
        console.error('Failed to load tiers:', error);
        this.error.set('We couldn\'t load plans right now. Please try again in a few moments.');
        this.loading.set(false);
      }
    });
  }

  private loadCurrentTier() {
    this.billingService.getSubscriptionInfo().subscribe({
      next: (subscription) => {
        this.currentTier.set(subscription?.tier || 'free');
      },
      error: (error) => {
        console.error('Failed to load current tier:', error);
        this.currentTier.set('free');
      }
    });
  }

  private loadUsageStats() {
    this.billingService.refreshUsageStats();
  }

  async upgrade(tierId: string) {
    if (!this.isAuthenticated) {
      this.router.navigate(['/auth'], { queryParams: { redirectTo: '/pricing' } });
      return;
    }

    if (tierId === 'free') {
      this.router.navigate(['/account']);
      return;
    }

    this.upgrading.set(tierId);

    try {
      const response = await this.billingService.createCheckoutSession(tierId).toPromise();

      if (response?.url) {
        window.location.href = response.url;
      } else {
        throw new Error('No checkout URL received');
      }
    } catch (error) {
      console.error('Failed to create checkout session:', error);
      this.error.set('Failed to start upgrade process. Please try again.');
      this.upgrading.set('');
    }
  }

  async renewCurrentPlan() {
    if (!this.isAuthenticated) {
      this.router.navigate(['/auth'], { queryParams: { redirectTo: '/pricing' } });
      return;
    }

    const confirmed = confirm(
      'Renew your subscription now?\n\n' +
      'This will:\n' +
      '• Charge you for a new billing period immediately\n' +
      '• Reset your token usage to 0\n' +
      '• Give you full quota for the next month\n\n' +
      'Continue?'
    );

    if (!confirmed) {
      return;
    }

    this.renewing.set(true);

    try {
      const result = await this.billingService.renewSubscription().toPromise();

      alert(result.message || 'Subscription renewed successfully! Your quota has been reset.');

      await this.loadTiers();
      if (this.isAuthenticated) {
        this.loadUsageStats();
      }
    } catch (error: any) {
      console.error('Failed to renew subscription:', error);
      alert('Failed to renew subscription: ' + (error.message || error.error?.message || 'Unknown error'));
    } finally {
      this.renewing.set(false);
    }
  }

  getTierDisplayName(tierId: string): string {
    const names: { [key: string]: string } = {
      'free': 'Free',
      'plus': 'Plus',
      'pro': 'Pro',
      'pro_plus': 'Pro Plus',
      'free_trial': 'Free Trial'
    };
    return names[tierId] || tierId;
  }

  formatTokens(tokens: number): string {
    return this.billingService.formatTokens(tokens);
  }

  formatModelList(models: string[]): string {
    if (!models || models.length === 0) {
      return '';
    }

    const friendlyNames: Record<string, string> = {
      'gpt-4o-mini': 'gpt-4o-mini',
      'gpt-4o': 'gpt-4o',
      'gpt-5.1': 'gpt-5.1',
      'gpt-5-mini': 'gpt-5-mini',
      'claude-3-haiku': 'claude-3-haiku',
      'claude-sonnet-4-5': 'claude-sonnet-4.5',
      'claude-opus-4-5': 'claude-opus-4.5',
      'gemini-2.0-flash-lite': 'gemini-2.0-flash-lite',
      'gemini-3-pro-preview': 'gemini-3-pro-preview',
      'gemini-2.5-flash': 'gemini-2.5-flash',
      'gemini-2.5-flash-lite': 'gemini-2.5-flash-lite'
    };

    return models.map(model => friendlyNames[model] ?? model).join(', ');
  }

  getCurrentTierPrice(): number {
    const current = this.currentTier();
    const tier = this.tiers().find(t => t.id === current);
    return tier?.price || 0;
  }

  viewUsage() {
    if (!this.isAuthenticated) {
      this.router.navigate(['/auth'], { queryParams: { redirectTo: '/usage' } });
      return;
    }
    this.router.navigate(['/usage']);
  }

  viewBilling() {
    if (!this.isAuthenticated) {
      this.router.navigate(['/auth'], { queryParams: { redirectTo: '/account' } });
      return;
    }
    this.router.navigate(['/account']);
  }

  trackByTier(index: number, tier: TierInfo): string {
    return tier.id;
  }
}
