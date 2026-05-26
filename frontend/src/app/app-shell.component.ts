import { Component, inject, signal, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterOutlet, Router } from '@angular/router';
import { AuthService } from './auth.service';
import { BillingService, UsageStats } from './services/billing.service';
import { Subscription } from 'rxjs';
import { UsageAlertsComponent } from './components/usage-alerts.component';
import { LogoComponent } from './shared/logo/logo.component';
import { environment } from '../environments/environment';

@Component({
  selector: 'app-shell',
  standalone: true,
  imports: [CommonModule, RouterOutlet, UsageAlertsComponent, LogoComponent],
  template: `
    <div class="app-shell">
      <!-- Main Header for all users except auth pages -->
      <header class="app-header" *ngIf="!isAuthPage()">
        <div class="header-content">
          <div class="app-title" (click)="navigateHome()">
            <app-logo [showTagline]="true" [tagline]="globalTagline" taglinePlacement="inline" [tight]="true"></app-logo>
          </div>
          
          <nav class="app-nav">
            <!-- Authenticated user navigation -->
            <button class="nav-button" (click)="toggleHistory()" title="Toggle Chat History" *ngIf="authService.isAuthenticated()">
              <img src="assets/history-icon.png" alt="History" class="history-icon">
            </button>
            <button class="new-chat-logo-button" (click)="newChat()" title="Start a new conversation" *ngIf="authService.isAuthenticated()">
              <img src="assets/new_chat_logo_top_right.png" alt="New Chat" class="new-chat-logo">
            </button>
            <button class="nav-button" (click)="navigateHome()" *ngIf="authService.isAuthenticated()">Chat</button>
            <button class="nav-button" (click)="navigateUsage()" *ngIf="authService.isAuthenticated()">Usage</button>
            
            <!-- Navigation for all users -->
            <button class="nav-button" (click)="navigateAbout()">About</button>
            <button class="nav-button" (click)="navigatePricing()">Pricing</button>

            <!-- Sign in button for unauthenticated users -->
            <button class="sign-in-button" (click)="navigateSignIn()" *ngIf="!authService.isAuthenticated()">Sign In</button>
            
            <!-- Usage indicator -->
            <div class="usage-indicator" *ngIf="authService.isAuthenticated() && usageStats()">
              <div class="usage-bar-mini">
                <div class="usage-fill-mini" 
                     [style.width.%]="usageStats()!.usagePercentage"
                     [class.warning]="usageStats()!.usagePercentage >= 80"
                     [class.danger]="usageStats()!.usagePercentage >= 95">
                </div>
              </div>
              <span class="usage-text-mini">{{ usageStats()!.usagePercentage.toFixed(0) }}%</span>
            </div>
            
            <div class="user-info" *ngIf="authService.currentUser()" (click)="navigateProfile()" title="Account settings">
              <div class="tier-badge-mini" *ngIf="usageStats()">
                {{ getTierDisplayName(getDisplayTier()) }}
              </div>
              <img 
                *ngIf="authService.currentUser()?.photoURL" 
                [src]="authService.currentUser()?.photoURL!" 
                [alt]="authService.currentUser()?.displayName || 'User'" 
                class="user-avatar"
              />
              <span class="user-name">{{ authService.currentUser()?.displayName || authService.currentUser()?.email }}</span>
            </div>
          </nav>
        </div>
      </header>

      <!-- Simple centered header for auth pages -->
      <header class="auth-header" *ngIf="isAuthPage()">
        <div class="auth-header-content">
          <app-logo [showTagline]="true" [tagline]="globalTagline" taglinePlacement="inline" [tight]="true" routerLink="/"></app-logo>
        </div>
      </header>

      <!-- Usage Alerts -->
      <div class="alerts-container" *ngIf="authService.isAuthenticated() && !isAuthPage()">
        <app-usage-alerts></app-usage-alerts>
      </div>

      <main class="app-main" [class.with-header]="!isAuthPage()" [class.with-auth-header]="isAuthPage()">
        <div class="loading-overlay" *ngIf="authService.isLoading()">
          <div class="loading-spinner">Loading...</div>
        </div>
        <router-outlet *ngIf="!authService.isLoading()"></router-outlet>
      </main>
    </div>
  `,
  styles: [`
    .app-shell {
      min-height: 100vh;
      display: flex;
      flex-direction: column;
    }

    .app-header {
      background: white;
      border-bottom: 1px solid #e1e5e9;
      box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
      z-index: 1000;
    }

    .header-content {
      max-width: 1200px;
      margin: 0 auto;
      padding: 0 1rem;
      display: flex;
      justify-content: space-between;
      align-items: center;
      height: 60px;
    }

    .app-title {
      display: inline-flex;
      cursor: pointer;
      transition: transform 0.2s ease;
    }

    .app-title:hover {
      transform: scale(1.01);
    }

    .app-title app-logo {
      display: inline-flex;
    }

    .app-nav {
      display: flex;
      align-items: center;
      gap: 1rem;
    }

    .nav-button {
      background: none;
      border: none;
      color: #4a5568;
      font-size: 1rem;
      font-weight: 500;
      padding: 0.5rem 1rem;
      border-radius: 6px;
      cursor: pointer;
      transition: all 0.2s ease;
    }

    .nav-button:hover {
      background: #f7fafc;
      color: #667eea;
    }

    .sign-in-button {
      background: white;
      color: #667eea;
      border: none;
      font-size: 1rem;
      font-weight: 500;
      padding: 0.5rem 1.5rem;
      border-radius: 6px;
      cursor: pointer;
      transition: all 0.2s ease;
      margin-left: 0.5rem;
    }

    .sign-in-button:hover {
      background: #f8fafc;
      transform: translateY(-1px);
    }

    .user-info {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      padding: 0.5rem;
      cursor: pointer;
      border-radius: 6px;
      transition: background-color 0.2s ease;
    }

    .user-info:hover {
      background: #f7fafc;
    }

    .user-avatar {
      width: 32px;
      height: 32px;
      border-radius: 50%;
      object-fit: cover;
    }

    .user-name {
      color: #4a5568;
      font-weight: 500;
      font-size: 0.9rem;
    }

    .new-chat-logo-button {
      background: none;
      border: none;
      padding: 0.75rem;
      border-radius: 8px;
      cursor: pointer;
      transition: all 0.3s ease;
      display: flex;
      align-items: center;
      justify-content: center;
      margin-right: 0.1rem;
    }

    .new-chat-logo-button:hover {
      background: #f7fafc;
      transform: scale(1.05);
    }

    .new-chat-logo {
      width: 32px;
      height: 32px;
      object-fit: contain;
    }

    .history-icon {
      width: 28px;
      height: 28px;
      object-fit: contain;
    }

    .usage-indicator {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      padding: 0.25rem 0.75rem;
      background: #f8fafc;
      border-radius: 16px;
      border: 1px solid #e2e8f0;
    }

    .usage-bar-mini {
      width: 40px;
      height: 6px;
      background: #e2e8f0;
      border-radius: 3px;
      overflow: hidden;
    }

    .usage-fill-mini {
      height: 100%;
      background: linear-gradient(90deg, #10b981, #059669);
      border-radius: 3px;
      transition: width 0.3s ease;
    }

    .usage-fill-mini.warning {
      background: linear-gradient(90deg, #f59e0b, #d97706);
    }

    .usage-fill-mini.danger {
      background: linear-gradient(90deg, #ef4444, #dc2626);
    }

    .usage-text-mini {
      font-size: 0.75rem;
      font-weight: 600;
      color: #475569;
    }

    .tier-badge-mini {
      background: linear-gradient(135deg, #3b82f6, #1d4ed8);
      color: white;
      padding: 0.125rem 0.5rem;
      border-radius: 10px;
      font-size: 0.625rem;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.05em;
    }

    .alerts-container {
      max-width: 1200px;
      margin: 0 auto;
      padding: 0 1rem;
    }

    .app-main {
      flex: 1;
      position: relative;
    }

    .app-main.with-header {
      padding-top: 0;
    }

    .app-main.with-auth-header {
      padding-top: 0;
    }

    .auth-header {
      background: white;
      padding: 2rem 0;
      text-align: center;
    }

    .auth-header-content {
      max-width: 1200px;
      margin: 0 auto;
      padding: 0 1rem;
      display: flex;
      justify-content: center;
      align-items: center;
    }

    .auth-header-content app-logo {
      display: inline-flex;
    }

    .loading-overlay {
      position: fixed;
      top: 0;
      left: 0;
      right: 0;
      bottom: 0;
      background: rgba(255, 255, 255, 0.9);
      display: flex;
      align-items: center;
      justify-content: center;
      z-index: 9999;
    }

    .loading-spinner {
      padding: 2rem;
      background: white;
      border-radius: 12px;
      box-shadow: 0 10px 30px rgba(0, 0, 0, 0.1);
      font-size: 1.1rem;
      color: #667eea;
    }

    @media (max-width: 768px) {
      .header-content {
        padding: 0 0.5rem;
      }

      .app-nav {
        gap: 0.5rem;
      }

      .nav-button {
        padding: 0.5rem;
        font-size: 0.9rem;
      }

      .user-name {
        display: none;
      }

      .new-chat-logo-button {
        padding: 0.5rem;
      }

      .new-chat-logo {
        width: 24px;
        height: 24px;
      }

      .history-icon {
        width: 20px;
        height: 20px;
      }
    }
  `]
})
export class AppShellComponent implements OnInit, OnDestroy {
  globalTagline = 'Private multi-model AI studio';
  private billingService = inject(BillingService);
  private subscriptions = new Subscription();
  private debug(...args: unknown[]): void {
    if (!environment.debugMode || !args.length) {
      return;
    }
    console.debug('[AppShell]', ...args);
  }
  
  usageStats = signal<UsageStats | null>(null);

  constructor(
    public authService: AuthService,
    private router: Router
  ) {}

  ngOnInit() {
    // Reduce GPU-intensive effects for Chromium-based browsers to avoid
    // compositing artifacts (backdrop-filter/large shadows) seen in Chrome.
    try {
      const ua = (typeof navigator !== 'undefined' ? navigator.userAgent : '') || '';
      const isSafari = /^((?!chrome|chromium|android|crios|edg).)*safari/i.test(ua);
      const isChromium = /(chrome|chromium|crios|edg)/i.test(ua) && !/opr|opera/i.test(ua);
      if (!isSafari && isChromium && typeof document !== 'undefined') {
        document.documentElement.classList.add('reduced-effects');
        this.debug('Reduced-effects mode enabled for Chromium');
      }
    } catch { /* ignore */ }

    // Subscribe to usage stats changes
    const usageStatsSubscription = this.billingService.usageStats$.subscribe(stats => {
      this.usageStats.set(stats);
    });
    this.subscriptions.add(usageStatsSubscription);
  }

  ngOnDestroy() {
    this.subscriptions.unsubscribe();
  }

  isAuthPage(): boolean {
    return this.router.url === '/auth';
  }

  navigateHome() {
    this.router.navigate(['/']);
  }

  navigateProfile() {
    this.debug('Account button clicked, navigating to account page');
    this.router.navigate(['/account']);
  }

  navigateAbout() {
    this.router.navigate(['/about']);
  }

  navigatePricing() {
    this.router.navigate(['/pricing']);
  }

  navigateUsage() {
    this.router.navigate(['/usage']);
  }

  navigateSignIn() {
    this.router.navigate(['/auth']);
  }

  navigateBilling() {
    this.router.navigate(['/account']);
  }

  getDisplayTier(): string {
    const stats = this.usageStats();
    if (!stats) {
      return 'free';
    }

    if (stats.freeTrial) {
      return 'free_trial';
    }

    return stats.currentTier;
  }

  getTierDisplayName(tier: string): string {
    const names: { [key: string]: string } = {
      'free': 'Free',
      'plus': 'Plus',
      'pro': 'Pro',
      'pro_plus': 'Pro+',
      'free_trial': 'Free Trial'
    };
    return names[tier] || tier;
  }

  newChat() {
    this.debug('New chat button clicked, current URL:', this.router.url);

    // Check if we're on the main chat page (could be /app for authenticated users)
    const isOnChatPage = this.router.url === '/app' || this.router.url === '/';

    if (!isOnChatPage) {
      this.debug('Not on chat page, navigating to /app first');
      this.router.navigate(['/app']).then((success) => {
        this.debug('Navigation to /app result:', success);
        if (success) {
          // After navigation, wait a bit then dispatch event to start new chat
          setTimeout(() => {
            this.debug('Dispatching newChat event after navigation');
            const event = new CustomEvent('newChat', {
              bubbles: true,
              detail: { source: 'header', timestamp: Date.now() }
            });
            window.dispatchEvent(event);
          }, 200);
        }
      }).catch((error) => {
        console.error('Navigation failed:', error);
      });
    } else {
      this.debug('Already on chat page, dispatching newChat event');
      // Already on chat page, just start new chat
      const event = new CustomEvent('newChat', {
        bubbles: true,
        detail: { source: 'header', timestamp: Date.now() }
      });
      window.dispatchEvent(event);
    }
  }

  toggleHistory() {
    // If not on main page, navigate to main page first, then show history
    if (this.router.url !== '/') {
      this.router.navigate(['/']).then(() => {
        // After navigation, wait a bit then dispatch event to show history
        setTimeout(() => {
          const event = new CustomEvent('toggleHistory', { 
            bubbles: true, 
            detail: { source: 'header', timestamp: Date.now(), forceShow: true } 
          });
          window.dispatchEvent(event);
        }, 100);
      });
    } else {
      // Dispatch a custom event to toggle history on main page
      const event = new CustomEvent('toggleHistory', { 
        bubbles: true, 
        detail: { source: 'header', timestamp: Date.now() } 
      });
      window.dispatchEvent(event);
    }
  }
}
