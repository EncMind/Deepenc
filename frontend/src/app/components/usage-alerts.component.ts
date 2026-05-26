import { Component, OnInit, OnDestroy, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { BillingService, UsageStats } from '../services/billing.service';
import { Subscription, interval } from 'rxjs';

@Component({
  selector: 'app-usage-alerts',
  standalone: true,
  imports: [CommonModule],
  template: `
    <!-- Critical Usage Alert (95%+) -->
    <div class="usage-alert critical" 
         *ngIf="usageStats() && getWarningLevel() === 'critical'"
         [@slideInDown]>
      <div class="alert-icon">🚨</div>
      <div class="alert-content">
        <div class="alert-title">Usage Limit Almost Reached</div>
        <div class="alert-message">
          You've used {{ usageStats()!.usagePercentage.toFixed(1) }}% of your monthly tokens. 
          API requests will be blocked when limit is reached.
        </div>
      </div>
      <div class="alert-actions">
        <button class="alert-btn primary" (click)="upgrade()">
          Upgrade Now
        </button>
        <button class="alert-btn dismiss" (click)="dismissAlert('critical')">
          ×
        </button>
      </div>
    </div>

    <!-- High Usage Alert (90%+) -->
    <div class="usage-alert high" 
         *ngIf="usageStats() && getWarningLevel() === 'high' && !isDismissed('high')"
         [@slideInDown]>
      <div class="alert-icon">⚠️</div>
      <div class="alert-content">
        <div class="alert-title">High Usage Warning</div>
        <div class="alert-message">
          You've used {{ usageStats()!.usagePercentage.toFixed(1) }}% of your monthly tokens.
          Consider upgrading to avoid service interruption.
        </div>
      </div>
      <div class="alert-actions">
        <button class="alert-btn secondary" (click)="viewUsage()">
          View Usage
        </button>
        <button class="alert-btn secondary" (click)="upgrade()">
          Upgrade
        </button>
        <button class="alert-btn dismiss" (click)="dismissAlert('high')">
          ×
        </button>
      </div>
    </div>

    <!-- Medium Usage Alert (80%+) -->
    <div class="usage-alert medium" 
         *ngIf="usageStats() && getWarningLevel() === 'medium' && !isDismissed('medium')"
         [@slideInDown]>
      <div class="alert-icon">💡</div>
      <div class="alert-content">
        <div class="alert-title">Usage Notification</div>
        <div class="alert-message">
          You've used {{ usageStats()!.usagePercentage.toFixed(1) }}% of your monthly tokens.
          {{ getRemainingDays() }} days left in billing period.
        </div>
      </div>
      <div class="alert-actions">
        <button class="alert-btn secondary" (click)="viewUsage()">
          View Details
        </button>
        <button class="alert-btn dismiss" (click)="dismissAlert('medium')">
          ×
        </button>
      </div>
    </div>

    <!-- Low Balance Alert for Paid Users -->
    <div class="usage-alert billing" 
         *ngIf="showBillingAlert() && !isDismissed('billing')"
         [@slideInDown]>
      <div class="alert-icon">💳</div>
      <div class="alert-content">
        <div class="alert-title">Billing Reminder</div>
        <div class="alert-message">
          Your next billing date is in {{ getDaysUntilBilling() }} days.
          Ensure your payment method is up to date.
        </div>
      </div>
      <div class="alert-actions">
        <button class="alert-btn secondary" (click)="manageBilling()">
          Manage Billing
        </button>
        <button class="alert-btn dismiss" (click)="dismissAlert('billing')">
          ×
        </button>
      </div>
    </div>
  `,
  styles: [`
    .usage-alert {
      display: flex;
      align-items: center;
      gap: 1rem;
      padding: 1rem 1.5rem;
      margin-bottom: 1rem;
      border-radius: 8px;
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
      animation: slideInDown 0.3s ease-out;
    }

    .usage-alert.critical {
      background: linear-gradient(135deg, #fef2f2, #fee2e2);
      border-left: 4px solid #dc2626;
    }

    .usage-alert.high {
      background: linear-gradient(135deg, #fffbeb, #fef3c7);
      border-left: 4px solid #f59e0b;
    }

    .usage-alert.medium {
      background: linear-gradient(135deg, #f0f9ff, #dbeafe);
      border-left: 4px solid #3b82f6;
    }

    .usage-alert.billing {
      background: linear-gradient(135deg, #f0fdf4, #dcfce7);
      border-left: 4px solid #10b981;
    }

    .alert-icon {
      font-size: 1.5rem;
      flex-shrink: 0;
    }

    .alert-content {
      flex: 1;
    }

    .alert-title {
      font-weight: 600;
      font-size: 0.95rem;
      color: #1a1a1a;
      margin-bottom: 0.25rem;
    }

    .alert-message {
      font-size: 0.875rem;
      color: #6b7280;
      line-height: 1.4;
    }

    .alert-actions {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      flex-shrink: 0;
    }

    .alert-btn {
      padding: 0.5rem 1rem;
      border-radius: 6px;
      font-size: 0.875rem;
      font-weight: 500;
      cursor: pointer;
      transition: all 0.2s ease;
      border: none;
    }

    .alert-btn.primary {
      background: linear-gradient(135deg, #3b82f6, #1d4ed8);
      color: white;
    }

    .alert-btn.primary:hover {
      transform: translateY(-1px);
      box-shadow: 0 4px 12px rgba(59, 130, 246, 0.4);
    }

    .alert-btn.secondary {
      background: white;
      color: #374151;
      border: 1px solid #d1d5db;
    }

    .alert-btn.secondary:hover {
      background: #f9fafb;
    }

    .alert-btn.dismiss {
      background: none;
      color: #9ca3af;
      padding: 0.25rem 0.5rem;
      font-size: 1.25rem;
      line-height: 1;
    }

    .alert-btn.dismiss:hover {
      color: #6b7280;
      background: rgba(0, 0, 0, 0.05);
    }

    @keyframes slideInDown {
      from {
        opacity: 0;
        transform: translateY(-20px);
      }
      to {
        opacity: 1;
        transform: translateY(0);
      }
    }

    @media (max-width: 768px) {
      .usage-alert {
        flex-direction: column;
        align-items: flex-start;
        gap: 0.75rem;
        padding: 1rem;
      }

      .alert-actions {
        width: 100%;
        justify-content: flex-end;
      }
    }
  `],
  animations: [
    // Add Angular animations if needed
  ]
})
export class UsageAlertsComponent implements OnInit, OnDestroy {
  private billingService = inject(BillingService);
  private router = inject(Router);

  usageStats = signal<UsageStats | null>(null);
  private usageSubscription?: Subscription;
  private intervalSubscription?: Subscription;
  private dismissedAlerts = new Set<string>();

  ngOnInit() {
    // Subscribe to usage stats updates
    this.usageSubscription = this.billingService.usageStats$.subscribe(stats => {
      this.usageStats.set(stats);
    });

    // Refresh usage stats periodically (every 5 minutes)
    this.intervalSubscription = interval(5 * 60 * 1000).subscribe(() => {
      this.billingService.refreshUsageStats();
    });

    // Initial load if not already loaded
    if (!this.usageStats()) {
      this.billingService.refreshUsageStats();
    }

    // Load dismissed alerts from localStorage
    this.loadDismissedAlerts();
  }

  ngOnDestroy() {
    // Clean up both subscriptions to prevent memory leaks
    this.usageSubscription?.unsubscribe();
    this.intervalSubscription?.unsubscribe();
  }

  getWarningLevel(): 'low' | 'medium' | 'high' | 'critical' {
    const stats = this.usageStats();
    if (!stats) return 'low';
    return this.billingService.getUsageWarningLevel(stats.usagePercentage);
  }

  showBillingAlert(): boolean {
    const stats = this.usageStats();
    if (!stats || stats.currentTier === 'free') return false;
    
    const daysUntilBilling = this.getDaysUntilBilling();
    return daysUntilBilling <= 3 && daysUntilBilling > 0;
  }

  getRemainingDays(): number {
    const stats = this.usageStats();
    if (!stats) return 0;
    return this.billingService.getDaysUntilReset(stats.billingPeriodEnd);
  }

  getDaysUntilBilling(): number {
    const stats = this.usageStats();
    if (!stats) return 0;
    return this.billingService.getDaysUntilReset(stats.billingPeriodEnd);
  }

  dismissAlert(type: string) {
    this.dismissedAlerts.add(type);
    this.saveDismissedAlerts();
  }

  isDismissed(type: string): boolean {
    return this.dismissedAlerts.has(type);
  }

  private loadDismissedAlerts() {
    const saved = localStorage.getItem('deepenc-dismissed-alerts');
    if (saved) {
      try {
        const alerts = JSON.parse(saved);
        this.dismissedAlerts = new Set(alerts);
      } catch (error) {
        console.error('Failed to load dismissed alerts:', error);
      }
    }
  }

  private saveDismissedAlerts() {
    const alerts = Array.from(this.dismissedAlerts);
    localStorage.setItem('deepenc-dismissed-alerts', JSON.stringify(alerts));
  }

  upgrade() {
    this.router.navigate(['/pricing']);
  }

  viewUsage() {
    this.router.navigate(['/usage']);
  }

  manageBilling() {
    this.router.navigate(['/account']);
  }
}
