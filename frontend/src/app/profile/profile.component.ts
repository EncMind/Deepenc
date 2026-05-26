import { Component, OnInit, OnDestroy, signal, NgZone } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { AuthService, User } from '../auth.service';
import { ChatService, ModelsResponse, ChosenModelsResponse, SetChosenModelsRequest } from '../chat.service';
import { 
  BillingService, 
  SubscriptionStatus, 
  SubscriptionInfo 
} from '../services/billing.service';
import { Subscription, distinctUntilChanged } from 'rxjs';
import { environment } from '../../environments/environment';

@Component({
  selector: 'app-profile',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="account-page">
      <section class="account-content">
        <div class="container">
          <div class="account-toolbar">
            <button class="logout-button" (click)="logout()">Sign Out</button>
          </div>
          <div class="card-grid">
            <article class="account-card" *ngIf="profile(); else profileSkeleton">
              <header>
                <h2>Personal Information</h2>
                <p>Set how your name appears across Deepenc experiences.</p>
              </header>
              <form class="form-layout" (ngSubmit)="updateProfile()" #profileForm="ngForm">
                <div class="form-group">
                  <label for="displayName">Display Name</label>
                  <input
                    type="text"
                    id="displayName"
                    name="displayName"
                    [(ngModel)]="displayName"
                    required
                    #nameInput="ngModel"
                  />
                </div>

                <div class="form-group">
                  <label for="email">Email</label>
                  <input
                    type="email"
                    id="email"
                    [value]="profile()?.email"
                    disabled
                    class="disabled-input"
                  />
                  <small class="help-text">Email cannot be changed</small>
                </div>

                <button type="submit" class="update-button" [disabled]="isUpdating() || profileForm.invalid">
                  <span *ngIf="!isUpdating()">Save changes</span>
                  <span *ngIf="isUpdating()">Updating...</span>
                </button>
              </form>
            </article>

            <ng-template #profileSkeleton>
              <article class="account-card skeleton-card">
                <header>
                  <h2>Personal Information</h2>
                </header>
                <p>Loading your profile details...</p>
              </article>
            </ng-template>

            <article class="account-card">
              <header>
                <h2>Model Preferences</h2>
                <p>Choose default models for each provider. You can still switch providers in any chat.</p>
              </header>

              <form class="form-layout" (ngSubmit)="updatePreferences()" #prefsForm="ngForm">
                <div class="form-group">
                  <label for="openaiModel">Chosen OpenAI Model</label>
                  <select id="openaiModel" name="openaiModel" [(ngModel)]="chosenModels.openai">
                    <option *ngFor="let model of availableModels()?.openai" [value]="model">{{ model }}</option>
                  </select>
                </div>

                <div class="form-group">
                  <label for="anthropicModel">Chosen Anthropic Model</label>
                  <select id="anthropicModel" name="anthropicModel" [(ngModel)]="chosenModels.anthropic">
                    <option *ngFor="let model of availableModels()?.anthropic" [value]="model">{{ model }}</option>
                  </select>
                </div>

                <div class="form-group">
                  <label for="geminiModel">Chosen Gemini Model</label>
                  <select id="geminiModel" name="geminiModel" [(ngModel)]="chosenModels.gemini">
                    <option *ngFor="let model of availableModels()?.gemini" [value]="model">{{ model }}</option>
                  </select>
                </div>

                <button type="submit" class="update-button" [disabled]="isUpdating() || prefsForm.invalid">
                  <span *ngIf="!isUpdating()">Save preferences</span>
                  <span *ngIf="isUpdating()">Updating...</span>
                </button>
              </form>
            </article>

            <article class="account-card wide">
              <header>
                <h2>Billing &amp; Subscription</h2>
                <p>Review your plan details and manage your subscription with Stripe.</p>
              </header>

              <div class="loading-state" *ngIf="billingLoading()">
                <div class="spinner"></div>
                <p>Loading billing information...</p>
              </div>

              <div class="plan-card" *ngIf="!billingLoading() && subscriptionInfo() as info">
                <div class="plan-header">
                  <div class="plan-copy">
                    <span class="label">Current Plan</span>
                    <div class="tier-display">
                      <span class="tier-name" [class]="'tier-' + (info.tier || 'free')">
                        {{ getTierDisplayName(info.tier || 'free') }}
                      </span>
                      <span class="plan-price" *ngIf="info.tier !== 'free' && info.tier !== 'free_trial'">
                        {{ '$' + getTierPrice(info.tier || '') }}/month
                      </span>
                      <span class="plan-price free" *ngIf="info.tier === 'free'">
                        Free forever
                      </span>
                      <span class="plan-price free" *ngIf="info.tier === 'free_trial'">
                        Complimentary · Ends {{ formatDate(info.freeTrial?.endAt) }}
                      </span>
                    </div>
                  </div>
                  <div class="plan-actions">
                    <button class="btn btn-secondary" (click)="viewPlans()">
                      View all plans
                    </button>
                    <button
                      class="btn btn-secondary"
                      (click)="manageBilling()"
                      *ngIf="info.tier !== 'free' && info.tier !== 'free_trial'"
                    >
                      Cancel subscription
                    </button>
                  </div>
                </div>

                <div class="subscription-status" *ngIf="subscriptionStatus() as status">
                  <div class="status-item">
                    <span class="label">Status</span>
                    <span class="value status" [class]="status.status || 'active'">
                      {{ info.tier === 'free_trial' ? 'Free Trial' : getStatusDisplayName(status.status || 'active') }}
                    </span>
                  </div>
                  <div class="status-item" *ngIf="info.tier === 'free_trial' && info.freeTrial?.endAt">
                    <span class="label">Trial ends</span>
                    <span class="value">{{ formatDate(info.freeTrial?.endAt) }}</span>
                  </div>
                  <div class="status-item" *ngIf="info.tier !== 'free_trial' && status.currentPeriodEnd">
                    <span class="label">Next billing</span>
                    <span class="value">{{ getNextBillingDate() }}</span>
                  </div>
                  <div class="status-item warning" *ngIf="status.cancelAtPeriodEnd">
                    <span class="label">Notice</span>
                    <span class="value">Renews until {{ getNextBillingDate() }}</span>
                  </div>
                </div>
              </div>

              <div class="error-state" *ngIf="billingError()">
                <div class="error-icon">⚠️</div>
                <h3>Unable to load billing information</h3>
                <p>{{ billingError() }}</p>
                <button class="btn btn-primary" (click)="loadBillingData()">Retry</button>
              </div>
            </article>
          </div>

          <div class="inline-feedback">
            <div class="loading" *ngIf="!profile()">Loading profile...</div>
            <div class="error-message" *ngIf="errorMessage()">
              {{ errorMessage() }}
            </div>
          </div>
        </div>
      </section>
    </div>
  `,
  styles: [`
    :host {
      display: block;
    }

    .account-page {
      --color-primary: #4f46e5;
      --color-primary-strong: #312e81;
      --color-accent: #14b8a6;
      --color-surface: rgba(255, 255, 255, 0.75);
      --color-surface-strong: rgba(255, 255, 255, 0.9);
      --color-border: rgba(148, 163, 184, 0.28);
      --color-divider: rgba(15, 23, 42, 0.08);
      --color-text: #0f172a;
      --color-muted: #64748b;
      --color-success: #16a34a;
      --color-warning: #f97316;
      --color-danger: #dc2626;
      font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      color: var(--color-text);
      background:
        radial-gradient(140% 140% at 15% 20%, rgba(99, 102, 241, 0.14), transparent 55%),
        radial-gradient(110% 110% at 85% 0%, rgba(14, 116, 144, 0.1), transparent 60%),
        #f8fafc;
      min-height: 100vh;
      line-height: 1.65;
    }

    section {
      padding: clamp(3rem, 6vw, 5rem) 0;
    }

    .container {
      width: min(1120px, calc(100% - 3rem));
      margin: 0 auto;
    }

    .logout-button {
      padding: 0.75rem 1.6rem;
      border-radius: 999px;
      border: 1px solid rgba(79, 70, 229, 0.2);
      background: rgba(255, 255, 255, 0.9);
      color: var(--color-primary-strong);
      font-weight: 600;
      cursor: pointer;
      transition: all 0.2s ease;
      box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.7);
    }

    .logout-button:hover {
      background: rgba(79, 70, 229, 0.08);
      border-color: rgba(79, 70, 229, 0.35);
      transform: translateY(-1px);
    }

    .account-toolbar {
      width: 930px;
      max-width: 100%;
      margin: 0 auto 2.5rem;
      padding: 0;
      display: flex;
      justify-content: flex-end;
    }

    .card-grid {
      display: grid;
      gap: 5rem;
      grid-template-columns: repeat(2, 380px);
      justify-content: center;
      align-items: stretch;
      padding: 0 2rem;
    }

    .account-card {
      background: var(--color-surface);
      border-radius: 26px;
      padding: 1.5rem;
      border: 1px solid rgba(255, 255, 255, 0.65);
      box-shadow:
        0 24px 35px rgba(15, 23, 42, 0.1),
        inset 0 1px 0 rgba(255, 255, 255, 0.7);
      display: flex;
      flex-direction: column;
      gap: 1.25rem;
      backdrop-filter: blur(18px);
      position: relative;
      overflow: hidden;
      width: 100%;
    }

    .account-card::after {
      content: '';
      position: absolute;
      inset: 0;
      background: linear-gradient(160deg, rgba(79, 70, 229, 0.08), transparent 60%);
      pointer-events: none;
    }

    .account-card header {
      position: relative;
      z-index: 1;
      display: grid;
      gap: 0.5rem;
    }

    .account-card header h2 {
      margin: 0;
      font-size: 1.6rem;
      font-weight: 600;
      color: var(--color-text);
    }

    .account-card header p {
      margin: 0;
      color: var(--color-muted);
      font-size: 0.98rem;
    }

    .account-card .form-layout {
      display: grid;
      gap: 1.25rem;
      position: relative;
      z-index: 1;
    }

    .account-card.wide {
      grid-column: 1 / -1;
      max-width: min(100%, 860px);
      justify-self: center;
      margin-left: 40px;
    }

    .account-card.skeleton-card::after {
      display: none;
    }

    .skeleton-card {
      color: var(--color-muted);
      min-height: 180px;
      display: grid;
      align-content: center;
      gap: 0.75rem;
    }

    .form-group {
      display: grid;
      gap: 0.5rem;
    }

    label {
      font-weight: 600;
      color: var(--color-text);
      font-size: 0.9rem;
    }

    input,
    select {
      padding: 0.8rem;
      border-radius: 12px;
      border: 1px solid var(--color-border);
      background: rgba(255, 255, 255, 0.9);
      font-size: 1rem;
      color: var(--color-text);
      transition: border-color 0.2s ease, box-shadow 0.2s ease;
    }

    input:focus,
    select:focus {
      border-color: rgba(79, 70, 229, 0.5);
      box-shadow: 0 0 0 4px rgba(99, 102, 241, 0.18);
      outline: none;
    }

    .disabled-input {
      color: #64748b;
      background: rgba(148, 163, 184, 0.12);
    }

    .help-text {
      color: var(--color-muted);
      font-size: 0.82rem;
    }

    .update-button {
      justify-self: flex-start;
      padding: 0.85rem 1.8rem;
      border-radius: 999px;
      border: 1px solid rgba(79, 70, 229, 0.28);
      background: linear-gradient(135deg, rgba(79, 70, 229, 0.16), rgba(14, 116, 144, 0.12));
      color: var(--color-primary-strong);
      font-weight: 600;
      cursor: pointer;
      transition: all 0.2s ease;
      display: inline-flex;
      align-items: center;
      gap: 0.5rem;
    }

    .update-button:hover:not(:disabled) {
      transform: translateY(-1px);
      box-shadow: 0 10px 20px rgba(79, 70, 229, 0.18);
    }

    .update-button:disabled {
      opacity: 0.6;
      cursor: not-allowed;
      box-shadow: none;
    }

    .loading-state {
      display: flex;
      align-items: center;
      gap: 1rem;
      padding: 1rem;
      color: var(--color-muted);
      background: rgba(255, 255, 255, 0.65);
      border-radius: 20px;
      border: 1px solid rgba(255, 255, 255, 0.6);
    }

    .spinner {
      width: 22px;
      height: 22px;
      border: 2px solid rgba(148, 163, 184, 0.4);
      border-top-color: rgba(79, 70, 229, 0.8);
      border-radius: 50%;
      animation: spin 1s linear infinite;
    }

    @keyframes spin {
      to {
        transform: rotate(360deg);
      }
    }

    .plan-card {
      position: relative;
      z-index: 1;
      display: grid;
      gap: 1.5rem;
      background: rgba(255, 255, 255, 0.78);
      border-radius: 22px;
      border: 1px solid rgba(255, 255, 255, 0.7);
      padding: clamp(1.75rem, 3vw, 2.25rem);
      box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.75);
    }

    .plan-header {
      display: flex;
      justify-content: space-between;
      gap: 1.5rem;
      flex-wrap: wrap;
      align-items: flex-start;
    }

    .plan-copy {
      display: grid;
      gap: 0.75rem;
    }

    .plan-copy .label {
      font-size: 0.8rem;
      text-transform: uppercase;
      letter-spacing: 0.1em;
      color: var(--color-muted);
      font-weight: 600;
    }

    .tier-display {
      display: grid;
      gap: 0.35rem;
    }

    .tier-name {
      display: inline-flex;
      align-items: center;
      padding: 0.4rem 0.85rem;
      border-radius: 999px;
      background: rgba(79, 70, 229, 0.14);
      color: var(--color-primary-strong);
      font-weight: 600;
      width: fit-content;
    }

    .tier-name.tier-free {
      background: rgba(148, 163, 184, 0.18);
      color: #1e293b;
    }

    .tier-name.tier-plus {
      background: rgba(59, 130, 246, 0.18);
      color: #1d4ed8;
    }

    .tier-name.tier-pro {
      background: rgba(74, 222, 128, 0.18);
      color: #166534;
    }

    .tier-name.tier-pro_plus {
      background: rgba(244, 114, 182, 0.22);
      color: #9d174d;
    }

    .tier-name.tier-free_trial {
      background: rgba(56, 189, 248, 0.2);
      color: #0e7490;
    }

    .plan-price {
      font-size: 0.95rem;
      color: var(--color-muted);
    }

    .plan-price.free {
      color: var(--color-success);
      font-weight: 600;
    }

    .plan-actions {
      display: flex;
      gap: 0.75rem;
      flex-wrap: wrap;
    }

    .btn {
      padding: 0.7rem 1.4rem;
      border-radius: 999px;
      font-weight: 600;
      font-size: 0.95rem;
      cursor: pointer;
      transition: all 0.2s ease;
      display: inline-flex;
      align-items: center;
      gap: 0.5rem;
    }

    .btn.btn-primary {
      background: linear-gradient(135deg, rgba(79, 70, 229, 0.2), rgba(14, 116, 144, 0.14));
      color: var(--color-primary-strong);
      border: 1px solid rgba(79, 70, 229, 0.32);
      box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.7);
    }

    .btn.btn-primary:hover {
      transform: translateY(-1px);
      box-shadow:
        0 12px 25px rgba(79, 70, 229, 0.18),
        inset 0 1px 0 rgba(255, 255, 255, 0.75);
    }

    .btn.btn-secondary {
      background: rgba(255, 255, 255, 0.85);
      color: var(--color-primary-strong);
      border: 1px solid rgba(79, 70, 229, 0.24);
      box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.8);
    }

    .btn.btn-secondary:hover {
      background: rgba(79, 70, 229, 0.08);
      border-color: rgba(79, 70, 229, 0.4);
      transform: translateY(-1px);
    }

    .btn:disabled {
      opacity: 0.6;
      cursor: not-allowed;
      transform: none;
      box-shadow: none;
    }

    .subscription-status {
      border-top: 1px solid rgba(148, 163, 184, 0.2);
      padding-top: 1.25rem;
      display: grid;
      gap: 0.9rem;
    }

    .status-item {
      display: flex;
      justify-content: space-between;
      gap: 1rem;
      flex-wrap: wrap;
    }

    .status-item .label {
      font-weight: 600;
      color: var(--color-muted);
      text-transform: uppercase;
      font-size: 0.78rem;
      letter-spacing: 0.08em;
    }

    .status-item .value {
      font-weight: 600;
      color: var(--color-text);
    }

    .status-item.warning .value {
      color: var(--color-warning);
    }

    .status-item .value.status.active {
      color: var(--color-success);
    }

    .status-item .value.status.canceled {
      color: var(--color-danger);
    }

    .status-item .value.status.past_due {
      color: var(--color-warning);
    }

    .error-state {
      text-align: center;
      padding: 2rem;
      background: rgba(255, 255, 255, 0.7);
      border-radius: 22px;
      border: 1px solid rgba(255, 255, 255, 0.7);
      box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.75);
      color: var(--color-muted);
      display: grid;
      gap: 0.75rem;
    }

    .error-state h3 {
      margin: 0;
      font-size: 1.2rem;
      color: var(--color-text);
    }

    .error-icon {
      font-size: 2rem;
    }

    .inline-feedback {
      margin-top: 2.5rem;
      display: grid;
      gap: 1rem;
      justify-items: center;
    }

    .loading {
      padding: 1rem 1.5rem;
      background: rgba(255, 255, 255, 0.6);
      border-radius: 16px;
      border: 1px solid rgba(255, 255, 255, 0.7);
      color: var(--color-muted);
    }

    .error-message {
      background: rgba(254, 226, 226, 0.85);
      color: var(--color-danger);
      padding: 1rem 1.5rem;
      border-radius: 16px;
      border: 1px solid rgba(248, 113, 113, 0.35);
    }

    @media (max-width: 720px) {
      .account-toolbar {
        width: 100%;
        padding: 0 1.5rem;
        margin: 0 0 2rem;
        justify-content: flex-start;
      }

      .plan-header {
        flex-direction: column;
        align-items: stretch;
      }

      .plan-actions {
        justify-content: flex-start;
      }
    }
  `]
})
export class ProfileComponent implements OnInit, OnDestroy {
  profile = signal<User | null>(null);
  isUpdating = signal(false);
  errorMessage = signal('');
  availableModels = signal<ModelsResponse | null>(null);

  // Billing signals
  billingLoading = signal(true);
  billingError = signal<string>('');
  subscriptionInfo = signal<SubscriptionInfo | null>(null);
  subscriptionStatus = signal<SubscriptionStatus | null>(null);

  displayName = '';
  chosenModels: SetChosenModelsRequest = {
    openai: '',
    anthropic: '',
    gemini: ''
  };
  private authSubscription?: Subscription;
  private debug(...args: unknown[]): void {
    if (!environment.debugMode || !args.length) {
      return;
    }
    // Avoid recursive calls; log directly
    console.debug('[ProfileComponent]', ...args);
  }

  constructor(
    private authService: AuthService, 
    private router: Router, 
    private chatService: ChatService, 
    private billingService: BillingService,
    private ngZone: NgZone
  ) {}

  ngOnInit() {
    this.debug('[ProfileComponent] ngOnInit called');
    // Load data in background, show UI immediately
    this.initializeModelSelectors();
    this.loadProfile();

    this.authSubscription = this.authService.user$
      .pipe(distinctUntilChanged((prev, curr) => prev?.uid === curr?.uid))
      .subscribe(user => {
        if (!user) {
          this.subscriptionInfo.set(null);
          this.subscriptionStatus.set(null);
          this.billingService.clearCachedSubscriptionInfo();
          this.billingService.clearCachedUsageStats();
          return;
        }

        this.loadBillingData();
      });
  }

  ngOnDestroy() {
    this.authSubscription?.unsubscribe();
  }

  private async initializeModelSelectors() {
    try {
      await this.loadModelsAsync();
      await this.loadChosenModelsAsync();
    } catch (error) {
      console.error('[ProfileComponent] Failed to initialize model selectors:', error);
    }
  }

  private async loadModelsAsync() {
    try {
      this.debug('[ProfileComponent] Loading available models...');
      const models = await this.chatService.getModels();
      this.debug('[ProfileComponent] Available models loaded:', models);
      
      // Run inside Angular zone to trigger change detection
      this.ngZone.run(() => {
        this.availableModels.set(models);
      });
    } catch (error) {
      console.error('[ProfileComponent] Failed to load models:', error);
      this.ngZone.run(() => {
        this.errorMessage.set('Failed to load available models');
      });
    }
  }

  private async loadChosenModelsAsync() {
    try {
      this.debug('[ProfileComponent] Loading chosen models...');
      const chosenModels = await this.chatService.getChosenModels();
      this.debug('[ProfileComponent] Chosen models loaded:', JSON.stringify(chosenModels));

      const available = this.availableModels();
      const openAIOptions = available?.openai || [];
      const anthropicOptions = available?.anthropic || [];
      const geminiOptions = available?.gemini || [];

      const resolved = {
        openai: this.resolveModelChoice(chosenModels.openai, openAIOptions),
        anthropic: this.resolveModelChoice(chosenModels.anthropic, anthropicOptions),
        gemini: this.resolveModelChoice(chosenModels.gemini, geminiOptions),
      };

      // Run inside Angular zone to trigger change detection
      this.ngZone.run(() => {
        this.chosenModels = resolved;
        this.debug('[ProfileComponent] chosenModels after resolution (in zone):', JSON.stringify(this.chosenModels));
      });
    } catch (error) {
      console.error('[ProfileComponent] Failed to load chosen models:', error);
      this.ngZone.run(() => {
        this.errorMessage.set('Failed to load chosen models');
        const available = this.availableModels();
        if (available) {
          this.chosenModels = {
            openai: this.resolveModelChoice(undefined, available.openai || []),
            anthropic: this.resolveModelChoice(undefined, available.anthropic || []),
            gemini: this.resolveModelChoice(undefined, available.gemini || []),
          };
        }
      });
    }
  }

  private resolveModelChoice(preferred: string | undefined, options: string[]): string {
    if (options.length === 0) {
      return preferred || '';
    }
    if (preferred && options.includes(preferred)) {
      return preferred;
    }
    return options[0];
  }

  private loadProfile() {
    const currentProfile = this.authService.getUserProfile();
    if (currentProfile) {
      this.profile.set(currentProfile);
      this.displayName = currentProfile.displayName;
    }
    
    // Subscribe to profile changes
    this.authService.profile$.subscribe(profile => {
      if (profile) {
        this.profile.set(profile);
        this.displayName = profile.displayName;
      }
    });
  }

  async updateProfile() {
    if (this.isUpdating()) return;

    this.isUpdating.set(true);
    this.errorMessage.set('');

    try {
      await this.authService.updateProfile({
        displayName: this.displayName
      });
    } catch (error: any) {
      this.errorMessage.set(error.message || 'Failed to update profile');
    } finally {
      this.isUpdating.set(false);
    }
  }

  async updatePreferences() {
    this.debug('[ProfileComponent] updatePreferences called');
    this.debug('[ProfileComponent] Current chosenModels:', JSON.stringify(this.chosenModels));
    
    if (this.isUpdating()) {
      this.debug('[ProfileComponent] Already updating, skipping...');
      return;
    }

    this.isUpdating.set(true);
    this.errorMessage.set('');

    try {
      this.debug('[ProfileComponent] About to call setChosenModels with:', JSON.stringify(this.chosenModels));
      await this.chatService.setChosenModels(this.chosenModels);
      this.debug('[ProfileComponent] Successfully updated chosen models:', this.chosenModels);
    } catch (error: any) {
      console.error('[ProfileComponent] Failed to update chosen models:', error);
      this.errorMessage.set(error.message || 'Failed to update chosen models');
    } finally {
      this.isUpdating.set(false);
    }
  }



  loadBillingData() {
    this.billingLoading.set(true);
    this.billingError.set('');

    // Load billing data
    Promise.all([
      this.billingService.getSubscriptionInfo().toPromise(),
      this.billingService.getSubscriptionStatus().toPromise()
    ]).then(([subscription, status]) => {
      // If no billing data exists yet, default to free tier
      if (!subscription || !status) {
        this.debug('No billing data found, defaulting to free tier');
        const now = Date.now();
        const defaultInfo: SubscriptionInfo = {
          tier: 'free',
          status: 'active',
          currentPeriodStart: now,
          currentPeriodEnd: now + 30 * 24 * 60 * 60 * 1000, // 30 days from now
          usage: {
            tokensUsed: 0,
            tokensLimit: 100000,
            usagePercentage: 0,
            providerBreakdown: {}
          },
          freeTrial: undefined
        };
        const defaultStatus: SubscriptionStatus = {
          status: 'active',
          tier: 'free',
          currentPeriodEnd: now + 30 * 24 * 60 * 60 * 1000,
          nextBillingDate: 0,
          cancelAtPeriodEnd: false,
          hasPaymentMethod: false,
          pastDue: false,
          freeTrialActive: false
        };
        this.subscriptionInfo.set(defaultInfo);
        this.subscriptionStatus.set(defaultStatus);
        this.billingLoading.set(false);
        return;
      }

      this.subscriptionInfo.set(subscription!);
      this.subscriptionStatus.set(status!);
      if (subscription?.freeTrial?.active && (!status || !status.tier || status.tier === 'free' || status.tier === 'free_trial')) {
        const trialInfo: SubscriptionInfo = {
          ...subscription!,
          tier: 'free_trial'
        };
        this.subscriptionInfo.set(trialInfo);
      }
      if (status?.freeTrialActive && (!status.tier || status.tier === 'free')) {
        const displayStatus: SubscriptionStatus = {
          ...status!,
          tier: 'free_trial'
        };
        this.subscriptionStatus.set(displayStatus);
      }
      this.billingLoading.set(false);
    }).catch((error) => {
      console.error('Failed to load billing data:', error);
      // Default to free tier on error instead of showing error message
      this.debug('Billing error, defaulting to free tier');
      const now = Date.now();
      const defaultInfo: SubscriptionInfo = {
        tier: 'free',
        status: 'active',
        currentPeriodStart: now,
        currentPeriodEnd: now + 30 * 24 * 60 * 60 * 1000, // 30 days from now
        usage: {
          tokensUsed: 0,
          tokensLimit: 100000,
          usagePercentage: 0,
          providerBreakdown: {}
        },
        freeTrial: undefined
      };
      const defaultStatus: SubscriptionStatus = {
        status: 'active',
        tier: 'free',
        currentPeriodEnd: now + 30 * 24 * 60 * 60 * 1000,
        nextBillingDate: 0,
        cancelAtPeriodEnd: false,
        hasPaymentMethod: false,
        pastDue: false,
        freeTrialActive: false
      };
      this.subscriptionInfo.set(defaultInfo);
      this.subscriptionStatus.set(defaultStatus);
      this.billingLoading.set(false);
    });
  }

  getTierDisplayName(tier: string): string {
    const names: { [key: string]: string } = {
      'free': 'Free',
      'plus': 'Plus',
      'pro': 'Pro',
      'pro_plus': 'Pro Plus',
      'free_trial': 'Free Trial'
    };
    return names[tier] || tier;
  }

  getTierPrice(tier: string): string {
    const prices: { [key: string]: string } = {
      'plus': '9.99',
      'pro': '19.99',
      'pro_plus': '49.99'
    };
    return prices[tier] || '0';
  }

  getStatusDisplayName(status: string): string {
    const names: { [key: string]: string } = {
      'active': 'Active',
      'canceled': 'Canceled',
      'past_due': 'Past Due',
      'incomplete': 'Incomplete'
    };
    return names[status] || status;
  }

  getNextBillingDate(): string {
    const status = this.subscriptionStatus();
    if (!status?.currentPeriodEnd) return '';
    return new Date(status.currentPeriodEnd).toLocaleDateString();
  }

  formatDate(timestamp?: number | null): string {
    if (!timestamp) {
      return '';
    }
    return new Date(timestamp).toLocaleDateString();
  }

  async manageBilling() {
    if (!confirm('Are you sure you want to cancel your subscription? You will retain access until the end of your current billing period.')) {
      return;
    }
    
    try {
      const response = await this.billingService.cancelSubscription().toPromise();
      if (response?.success) {
        alert(response.message || 'Subscription has been scheduled for cancellation.');
        // Refresh the subscription data to show updated status
        this.loadBillingData();
      }
    } catch (error) {
      console.error('Failed to cancel subscription:', error);
      this.billingError.set('Failed to cancel subscription. Please try again.');
    }
  }

  viewPlans() {
    this.router.navigate(['/pricing']);
  }

  async logout() {
    try {
      await this.authService.logout();
      this.router.navigate(['/auth']);
    } catch (error) {
      console.error('Logout error:', error);
      this.errorMessage.set('Failed to sign out. Please try again.');
    }
  }
}
