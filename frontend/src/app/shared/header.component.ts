import { Component, OnInit, OnDestroy, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule, Router } from '@angular/router';
import { Subscription } from 'rxjs';
import { distinctUntilChanged, withLatestFrom } from 'rxjs/operators';
import { AuthService } from '../auth.service';
import { BillingService } from '../services/billing.service';
import { LogoComponent } from './logo/logo.component';

@Component({
  selector: 'app-header',
  standalone: true,
  imports: [CommonModule, RouterModule, LogoComponent],
  template: `
    <header class="header">
      <div class="container">
        <div class="nav-content">
          <!-- Logo and Brand -->
          <app-logo
            class="brand"
            [routerLink]="getHomeRoute()"
            theme="dark"
            taglinePlacement="inline"
            [tagline]="tagline"
            [tight]="true">
          </app-logo>
          
          <!-- Navigation Links -->
          <nav class="nav-links">
            <a routerLink="/about" class="nav-link" routerLinkActive="active">About</a>
            <a routerLink="/pricing" class="nav-link" routerLinkActive="active">Pricing</a>
          </nav>

          <!-- About Button (always visible) -->
          <div class="about-button-container">
            <a routerLink="/about" class="about-button" routerLinkActive="active">About</a>
          </div>
          
          <!-- Auth Section -->
          <div class="auth-section">
            <ng-container *ngIf="isAuthenticated(); else unauthenticatedTemplate">
              <!-- Authenticated User -->
              <div class="user-menu">
                <span class="user-email">{{ userEmail() }}</span>
                <div class="user-actions">
                  <a routerLink="/account" class="nav-link">Account</a>
                  <a routerLink="/usage" class="nav-link" *ngIf="!isFreeUser()">Usage</a>
                  <button class="logout-button" (click)="logout()">Sign Out</button>
                </div>
              </div>
            </ng-container>
            
            <ng-template #unauthenticatedTemplate>
              <!-- Unauthenticated User -->
              <div class="auth-buttons">
                <a routerLink="/auth" class="auth-link">Sign In</a>
                <a routerLink="/auth" class="auth-button">Get Started</a>
              </div>
            </ng-template>
          </div>
        </div>
      </div>
    </header>
  `,
  styles: [`
    .header {
      background: white;
      border-bottom: 1px solid #e2e8f0;
      position: sticky;
      top: 0;
      z-index: 100;
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
    }

    .container {
      max-width: 1200px;
      margin: 0 auto;
      padding: 0 1rem;
    }

    .nav-content {
      display: flex;
      align-items: center;
      justify-content: space-between;
      height: 64px;
      gap: 2rem;
    }

    .brand {
      flex-shrink: 0;
      display: inline-flex;
      align-items: center;
    }

    .nav-links {
      display: flex;
      align-items: center;
      gap: 2rem;
      flex: 1;
      justify-content: center;
    }

    .nav-link {
      color: #4b5563;
      text-decoration: none;
      font-weight: 500;
      padding: 0.5rem 1rem;
      border-radius: 6px;
      transition: all 0.2s ease;
    }

    .nav-link:hover {
      color: #667eea;
      background: #f8fafc;
    }

    .nav-link.active {
      color: #667eea;
      background: #e0e7ff;
    }

    .about-button-container {
      flex-shrink: 0;
    }

    .about-button {
      background: #667eea;
      color: white;
      text-decoration: none;
      font-weight: 600;
      padding: 0.5rem 1rem;
      border-radius: 6px;
      transition: all 0.2s ease;
      font-size: 0.875rem;
    }

    .about-button:hover {
      background: #5a6fd8;
      transform: translateY(-1px);
    }

    .about-button.active {
      background: #4c51bf;
    }

    .auth-section {
      flex-shrink: 0;
    }

    .auth-buttons {
      display: flex;
      align-items: center;
      gap: 1rem;
    }

    .auth-link {
      color: #4b5563;
      text-decoration: none;
      font-weight: 500;
      padding: 0.5rem 1rem;
      border-radius: 6px;
      transition: all 0.2s ease;
    }

    .auth-link:hover {
      color: #667eea;
      background: #f8fafc;
    }

    .auth-button {
      background: #667eea;
      color: white;
      text-decoration: none;
      font-weight: 600;
      padding: 0.5rem 1.5rem;
      border-radius: 6px;
      transition: all 0.2s ease;
    }

    .auth-button:hover {
      background: #5a6fd8;
      transform: translateY(-1px);
    }

    .user-menu {
      display: flex;
      align-items: center;
      gap: 1rem;
    }

    .user-email {
      color: #6b7280;
      font-size: 0.875rem;
      font-weight: 500;
    }

    .user-actions {
      display: flex;
      align-items: center;
      gap: 0.5rem;
    }

    .logout-button {
      background: none;
      border: 1px solid #d1d5db;
      color: #6b7280;
      font-weight: 500;
      padding: 0.5rem 1rem;
      border-radius: 6px;
      cursor: pointer;
      transition: all 0.2s ease;
    }

    .logout-button:hover {
      background: #f9fafb;
      border-color: #9ca3af;
      color: #374151;
    }

    /* Responsive Design */
    @media (max-width: 768px) {
      .nav-content {
        gap: 1rem;
      }

      .brand .logo-tagline {
        display: none;
      }

      .nav-links {
        display: none;
      }

      .about-button {
        padding: 0.375rem 0.75rem;
        font-size: 0.875rem;
      }

      .user-actions {
        flex-direction: column;
        gap: 0.25rem;
      }

      .user-email {
        display: none;
      }

      .auth-buttons {
        flex-direction: column;
        gap: 0.5rem;
      }
    }

    @media (max-width: 480px) {
      .container {
        padding: 0 0.5rem;
      }

      .brand .logo-name {
        font-size: 1.35rem;
      }

      .auth-button,
      .auth-link {
        padding: 0.375rem 1rem;
        font-size: 0.875rem;
      }
    }
  `]
})
export class HeaderComponent implements OnInit, OnDestroy {
  tagline = 'Private multi-model AI studio';
  isAuthenticated = signal<boolean>(false);
  userEmail = signal<string>('');
  isFreeUser = signal<boolean>(true); // Default to true (hide usage for free users)
  private userSubscription?: Subscription;
  private subscriptionInfoSubscription?: Subscription;
  private currentUserId: string | null = null;

  constructor(
    private authService: AuthService,
    private billingService: BillingService,
    private router: Router
  ) {}

  ngOnInit() {
    // Track auth state with deduplication
    this.userSubscription = this.authService.user$.pipe(
      distinctUntilChanged((prev, curr) => prev?.uid === curr?.uid)
    ).subscribe(user => {
      this.currentUserId = user?.uid || null;
      this.isAuthenticated.set(!!user);
      this.userEmail.set(user?.email || '');

      if (user) {
        this.billingService.refreshSubscriptionInfo();
      } else {
        this.isFreeUser.set(true);
        this.billingService.clearCachedSubscriptionInfo();
        this.billingService.clearCachedUsageStats();
      }
    });

    // React to subscription info updates, guarding against stale emissions
    this.subscriptionInfoSubscription = this.billingService.subscriptionInfo$.pipe(
      withLatestFrom(this.authService.user$)
    ).subscribe(([info, user]) => {
      if (!user || user.uid !== this.currentUserId) {
        // Either not authenticated or the emission belongs to a previous user session
        return;
      }

      if (!info) {
        this.isFreeUser.set(true);
        return;
      }

      this.isFreeUser.set(!info.tier || info.tier === 'free');
    });
  }

  ngOnDestroy() {
    this.userSubscription?.unsubscribe();
    this.subscriptionInfoSubscription?.unsubscribe();
  }

  getHomeRoute(): string {
    return this.isAuthenticated() ? '/app' : '/';
  }

  async logout() {
    try {
      await this.authService.logout();
      this.billingService.clearCachedSubscriptionInfo();
      this.billingService.clearCachedUsageStats();
      this.isFreeUser.set(true);
      this.router.navigate(['/']);
    } catch (error) {
      console.error('Logout error:', error);
    }
  }
}
