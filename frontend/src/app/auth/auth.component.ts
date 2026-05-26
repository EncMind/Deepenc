import { Component, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { AuthService } from '../auth.service';
import { LogoComponent } from '../shared/logo/logo.component';

@Component({
  selector: 'app-auth',
  standalone: true,
  imports: [CommonModule, FormsModule, LogoComponent],
  template: `
    <div class="auth-page">
      <div class="auth-layout">
        <aside class="auth-aside">
            <div class="auth-aside-content">
              <app-logo theme="light" size="md" [showTagline]="true" [tagline]="brandTagline" taglinePlacement="stacked" [tight]="true"></app-logo>
            <h2>Confidence for independent creators</h2>
            <p>
              Ship client work, research, and personal projects with enterprise-grade privacy, all inside a single multi-model studio.
            </p>
            <ul class="aside-list">
              <li>Secure prompts in a hardware-isolated enclave.</li>
              <li>Switch between GPT, Claude, and Gemini without losing context.</li>
              <li>Side-by-side model responses&mdash;choose the best result.</li>
            </ul>
            <div class="aside-meta">
              <span>Azure Confidential Computing</span>
              <span>Zero-knowledge architecture</span>
            </div>
          </div>
        </aside>

        <section class="auth-panel">
          <div class="auth-card">
            <ng-container *ngIf="showEmailVerification(); else authStates">
              <div class="panel-header">
                <span class="panel-kicker">Check your inbox</span>
                <h1>Verify your email</h1>
                <p>We sent a secure sign-on link to <strong>{{ verificationEmail() }}</strong>.</p>
              </div>

              <div class="verification-card">
                <div class="verification-icon">📧</div>
                <p>Open the message and confirm your email to activate your private workspace.</p>
                <p class="verification-note">
                  When you're done, come back here and click the button below.
                </p>
              </div>

              <div class="alert success" *ngIf="statusMessage()">
                {{ statusMessage() }}
              </div>
              <div class="alert error" *ngIf="!statusMessage() && errorMessage()">
                {{ errorMessage() }}
              </div>

              <button
                type="button"
                class="btn btn-primary"
                [disabled]="isLoading()"
                (click)="checkEmailVerification()">
                <span *ngIf="!isLoading()">I've verified my email</span>
                <span *ngIf="isLoading()" class="btn-spinner">Checking...</span>
              </button>

              <div class="panel-footer">
                <p>
                  Didn't get the email?
                  <button type="button" class="btn-link" (click)="resendEmail()" [disabled]="isLoading()">
                    Resend link
                  </button>
                </p>
                <button type="button" class="btn-link subtle" (click)="backToLogin()">
                  Back to sign in
                </button>
              </div>
            </ng-container>

            <ng-template #authStates>
              <ng-container *ngIf="showForgotPassword(); else mainAuth">
                <div class="panel-header">
                  <span class="panel-kicker">Reset access</span>
                  <h1>Forgot your password?</h1>
                  <p>Enter the email you use with Deepenc. We'll send a secure reset link right away.</p>
                </div>

                <div class="alert success" *ngIf="statusMessage()">
                  {{ statusMessage() }}
                </div>
                <div class="alert error" *ngIf="!statusMessage() && errorMessage()">
                  {{ errorMessage() }}
                </div>

                <div class="form-field">
                  <label for="resetEmail">Email address</label>
                  <input
                    id="resetEmail"
                    type="email"
                    name="resetEmail"
                    [(ngModel)]="resetEmail"
                    required
                    email
                    placeholder="you@example.com"
                    class="form-input"
                  />
                </div>

                <button
                  type="button"
                  class="btn btn-primary"
                  [disabled]="isLoading() || !resetEmail"
                  (click)="sendPasswordReset()">
                  <span *ngIf="!isLoading()">Send reset email</span>
                  <span *ngIf="isLoading()" class="btn-spinner">Sending...</span>
                </button>

                <div class="panel-footer">
                  <button type="button" class="btn-link subtle" (click)="backToLogin()">
                    Back to sign in
                  </button>
                </div>
              </ng-container>
            </ng-template>

            <ng-template #mainAuth>
              <div class="panel-header">
                <span class="panel-kicker">{{ isLoginMode() ? 'Welcome back' : 'Create your workspace' }}</span>
                <h1>{{ isLoginMode() ? 'Sign in to Deepenc' : 'Join Deepenc' }}</h1>
                <p>
                  {{ isLoginMode()
                    ? 'Access your private, multi-model assistant in seconds.'
                    : 'Set up a secure, zero-knowledge environment for your independent projects.' }}
                </p>
              </div>

              <div class="alert success" *ngIf="statusMessage()">
                {{ statusMessage() }}
              </div>
              <div class="alert error" *ngIf="!statusMessage() && errorMessage()">
                {{ errorMessage() }}
              </div>

              <form class="auth-form" (ngSubmit)="onSubmit()" #authForm="ngForm">
                <div class="form-field">
                  <label for="email">Email</label>
                  <input
                    type="email"
                    id="email"
                    name="email"
                    [(ngModel)]="email"
                    required
                    email
                    #emailInput="ngModel"
                    [class.input-error]="emailInput.invalid && emailInput.touched"
                    class="form-input"
                    placeholder="you@example.com"
                  />
                  <div class="field-error" *ngIf="emailInput.invalid && emailInput.touched">
                    Enter a valid email address.
                  </div>
                </div>

                <div class="form-field" *ngIf="!isLoginMode()">
                  <label for="displayName">Display name</label>
                  <input
                    type="text"
                    id="displayName"
                    name="displayName"
                    [(ngModel)]="displayName"
                    [required]="!isLoginMode()"
                    #nameInput="ngModel"
                    [class.input-error]="nameInput.invalid && nameInput.touched"
                    class="form-input"
                    placeholder="How should we address you?"
                  />
                  <div class="field-error" *ngIf="nameInput.invalid && nameInput.touched">
                    Display name is required.
                  </div>
                </div>

                <div class="form-field">
                  <label for="password">Password</label>
                  <input
                    type="password"
                    id="password"
                    name="password"
                    [(ngModel)]="password"
                    required
                    minlength="6"
                    #passwordInput="ngModel"
                    [class.input-error]="passwordInput.invalid && passwordInput.touched"
                    class="form-input"
                    placeholder="Enter your password"
                  />
                  <div class="field-error" *ngIf="passwordInput.invalid && passwordInput.touched">
                    Password must be at least 6 characters.
                  </div>
                </div>

                <div class="form-field" *ngIf="!isLoginMode()">
                  <label for="confirmPassword">Confirm password</label>
                  <input
                    type="password"
                    id="confirmPassword"
                    name="confirmPassword"
                    [(ngModel)]="confirmPassword"
                    [required]="!isLoginMode()"
                    #confirmInput="ngModel"
                    [class.input-error]="(confirmInput.invalid && confirmInput.touched) || (password !== confirmPassword && confirmInput.touched)"
                    class="form-input"
                    placeholder="Repeat your password"
                  />
                  <div class="field-error" *ngIf="confirmInput.touched && password !== confirmPassword">
                    Passwords must match.
                  </div>
                </div>

                <div class="terms" *ngIf="!isLoginMode()">
                  <label class="terms-label">
                    <input
                      type="checkbox"
                      name="termsAgreed"
                      [(ngModel)]="termsAgreed"
                      [required]="!isLoginMode()"
                      #termsInput="ngModel"
                    />
                    <span>
                      I agree to Deepenc's
                      <a href="/terms" target="_blank" rel="noopener" class="terms-link">Terms</a>
                      and
                      <a href="/privacy" target="_blank" rel="noopener" class="terms-link">Acceptable Use Policy</a>
                      and confirm I'm at least 18 years old.
                    </span>
                  </label>
                  <div class="field-error" *ngIf="termsInput.invalid && termsInput.touched">
                    You must accept the terms to continue.
                  </div>
                </div>

                <button
                  type="submit"
                  class="btn btn-primary"
                  [disabled]="authForm.invalid || isLoading() || (!isLoginMode() && password !== confirmPassword) || (!isLoginMode() && !termsAgreed)"
                >
                  <span *ngIf="!isLoading()" class="btn-label">{{ isLoginMode() ? 'Sign in' : 'Create account' }}</span>
                  <span *ngIf="isLoading()" class="btn-spinner">Loading...</span>
                </button>
              </form>

              <div class="form-separator">
                <span>or</span>
              </div>

              <button
                type="button"
                class="btn btn-ghost btn-google"
                [disabled]="isLoading()"
                (click)="signInWithGoogle()">
                <svg width="18" height="18" viewBox="0 0 18 18" aria-hidden="true" focusable="false">
                  <path fill="#4285F4" d="M16.51 8H8.98v3h4.3c-.18 1-.74 1.48-1.6 2.04v2.01h2.6a7.8 7.8 0 0 0 2.38-5.88c0-.57-.05-.66-.15-1.18z"/>
                  <path fill="#34A853" d="M8.98 17c2.16 0 3.97-.72 5.3-1.94l-2.6-2.04a4.8 4.8 0 0 1-2.7.75c-2.08 0-3.84-1.4-4.48-3.29H1.96v2.09A7.86 7.86 0 0 0 8.98 17z"/>
                  <path fill="#FBBC05" d="M4.5 10.48A4.59 4.59 0 0 1 4.25 9c0-.51.09-1.02.25-1.48V5.43H1.96a7.86 7.86 0 0 0 0 7.14l2.54-2.09z"/>
                  <path fill="#EB4335" d="M8.98 4.13c1.17 0 2.23.4 3.06 1.2l2.3-2.28A7.86 7.86 0 0 0 8.98 1a7.86 7.86 0 0 0-7.02 4.43l2.54 2.09c.63-1.89 2.39-3.29 4.48-3.29z"/>
                </svg>
                <span *ngIf="!isLoading()">Continue with Google</span>
                <span *ngIf="isLoading()" class="btn-spinner">Signing in...</span>
              </button>

              <div class="panel-footer">
                <p>
                  {{ isLoginMode() ? "Don't have an account?" : "Already using Deepenc?" }}
                  <button type="button" class="btn-link" (click)="toggleMode()">
                    {{ isLoginMode() ? 'Create one' : 'Sign in' }}
                  </button>
                </p>
                <p *ngIf="isLoginMode()">
                  Forgot your password?
                  <button type="button" class="btn-link" (click)="showForgotPasswordScreen()">
                    Reset it
                  </button>
                </p>
              </div>
            </ng-template>
          </div>
        </section>
      </div>
    </div>
  `,
  styles: [`
    :host {
      display: block;
    }

    .auth-page {
      min-height: 100vh;
      padding: clamp(2rem, 5vw, 3.5rem) clamp(1.5rem, 4vw, 3rem);
      background:
        radial-gradient(120% 120% at 15% 20%, rgba(99, 102, 241, 0.16), transparent 55%),
        radial-gradient(100% 100% at 85% 0%, rgba(14, 165, 233, 0.16), transparent 60%),
        #0f172a;
      display: flex;
      align-items: center;
      justify-content: center;
    }

    .auth-layout {
      width: min(1100px, 100%);
      display: grid;
      gap: clamp(2rem, 5vw, 3rem);
      grid-template-columns: minmax(0, 480px) minmax(0, 420px);
      align-items: stretch;
    }

    .auth-aside {
      background:
        linear-gradient(160deg, rgba(37, 56, 124, 0.92) 0%, rgba(24, 39, 95, 0.9) 45%, rgba(17, 24, 55, 0.88) 100%);
      border-radius: 28px;
      padding: clamp(2rem, 5vw, 2.8rem);
      border: 1px solid rgba(148, 163, 184, 0.22);
      color: #f8fafc;
      display: flex;
      flex-direction: column;
      gap: 1.75rem;
      box-shadow:
        0 25px 65px rgba(15, 23, 42, 0.45),
        inset 0 1px 0 rgba(255, 255, 255, 0.06);
      position: relative;
      overflow: hidden;
      backdrop-filter: blur(22px);
    }

    .auth-aside::after {
      content: '';
      position: absolute;
      inset: 0;
      background:
        radial-gradient(115% 115% at 20% 15%, rgba(129, 140, 248, 0.32), transparent 70%),
        radial-gradient(120% 120% at 80% 20%, rgba(56, 189, 248, 0.28), transparent 72%);
      opacity: 0.75;
      pointer-events: none;
    }

    .auth-aside-content {
      position: relative;
      z-index: 1;
      display: grid;
      gap: 1.75rem;
      color: rgba(248, 250, 255, 0.96);
      text-shadow: 0 6px 18px rgba(15, 23, 42, 0.3);
    }

    .auth-aside-content app-logo {
      display: inline-flex;
    }

    .auth-aside h2 {
      font-size: clamp(1.9rem, 3.4vw, 2.4rem);
      font-weight: 700;
      letter-spacing: -0.015em;
      margin: 0;
      color: #ffffff !important;
      text-shadow: 0 8px 22px rgba(15, 23, 42, 0.3);
    }

    .auth-aside p {
      margin: 0;
      color: rgba(255, 255, 255, 0.94) !important;
      line-height: 1.7;
      font-size: 1rem;
      text-shadow: 0 6px 18px rgba(15, 23, 42, 0.28);
    }

    .aside-list {
      list-style: none;
      margin: 0;
      padding: 0;
      display: grid;
      gap: 1rem;
    }

    .aside-list li {
      position: relative;
      padding-left: 1.6rem;
      font-size: 0.98rem;
      color: rgba(255, 255, 255, 0.94) !important;
      line-height: 1.6;
      text-shadow: 0 6px 18px rgba(15, 23, 42, 0.28);
    }

    .aside-list li::before {
      content: '';
      position: absolute;
      left: 0;
      top: 0.55rem;
      width: 10px;
      height: 10px;
      border-radius: 50%;
      background: linear-gradient(135deg, rgba(224, 231, 255, 1), rgba(148, 197, 255, 1));
      box-shadow: 0 0 18px rgba(148, 163, 255, 0.6);
    }

    .aside-meta {
      margin-top: auto;
      display: flex;
      flex-wrap: wrap;
      gap: 0.75rem;
    }

    .aside-meta span {
      padding: 0.35rem 0.9rem;
      border-radius: 999px;
      border: 1px solid rgba(217, 226, 255, 0.7);
      background: rgba(241, 248, 255, 0.28);
      font-size: 0.8rem;
      font-weight: 600;
      letter-spacing: 0.08em;
      text-transform: uppercase;
      color: rgba(255, 255, 255, 0.95) !important;
      text-shadow: 0 6px 18px rgba(15, 23, 42, 0.28);
    }

    .auth-panel {
      display: flex;
      align-items: center;
      justify-content: center;
    }

    .auth-card {
      width: min(420px, 100%);
      background: rgba(255, 255, 255, 0.94);
      border-radius: 24px;
      padding: clamp(2rem, 5vw, 2.6rem);
      box-shadow:
        0 25px 45px rgba(15, 23, 42, 0.18),
        inset 0 1px 0 rgba(255, 255, 255, 0.9);
      border: 1px solid rgba(148, 163, 184, 0.2);
      display: grid;
      gap: 1.75rem;
    }

    .panel-header {
      display: grid;
      gap: 0.75rem;
      text-align: left;
    }

    .panel-kicker {
      font-size: 0.78rem;
      font-weight: 600;
      letter-spacing: 0.18em;
      text-transform: uppercase;
      color: rgba(79, 70, 229, 0.8);
    }

    .panel-header h1 {
      font-size: clamp(1.6rem, 3vw, 2.1rem);
      font-weight: 700;
      color: #0f172a;
      margin: 0;
    }

    .panel-header p {
      margin: 0;
      color: #475569;
      line-height: 1.6;
      font-size: 0.98rem;
    }

    .verification-card {
      background: rgba(79, 70, 229, 0.08);
      border-radius: 20px;
      padding: 1.8rem;
      border: 1px solid rgba(79, 70, 229, 0.18);
      text-align: center;
      display: grid;
      gap: 0.75rem;
      color: #4338ca;
    }

    .verification-icon {
      font-size: 2.4rem;
    }

    .verification-card p {
      margin: 0;
      color: #3730a3;
    }

    .verification-note {
      font-size: 0.9rem;
      color: rgba(67, 56, 202, 0.78);
    }

    .alert {
      padding: 0.9rem 1.1rem;
      border-radius: 16px;
      font-size: 0.92rem;
      line-height: 1.5;
    }

    .alert.error {
      background: rgba(248, 113, 113, 0.12);
      border: 1px solid rgba(248, 113, 113, 0.32);
      color: #b91c1c;
    }

    .alert.success {
      background: rgba(74, 222, 128, 0.12);
      border: 1px solid rgba(74, 222, 128, 0.32);
      color: #047857;
    }

    .auth-form {
      display: grid;
      gap: 1.2rem;
    }

    .form-field {
      display: grid;
      gap: 0.45rem;
    }

    .form-field label {
      font-size: 0.85rem;
      font-weight: 600;
      color: #1e293b;
      letter-spacing: 0.02em;
    }

    .form-input {
      border-radius: 12px;
      border: 1px solid rgba(148, 163, 184, 0.4);
      background: rgba(255, 255, 255, 0.9);
      padding: 0.85rem 1rem;
      font-size: 0.98rem;
      color: #0f172a;
      transition: border-color 0.2s ease, box-shadow 0.2s ease;
    }

    .form-input:focus {
      outline: none;
      border-color: rgba(79, 70, 229, 0.55);
      box-shadow: 0 0 0 4px rgba(79, 70, 229, 0.12);
    }

    .form-input.input-error {
      border-color: rgba(239, 68, 68, 0.75);
      box-shadow: 0 0 0 4px rgba(239, 68, 68, 0.15);
    }

    .field-error {
      font-size: 0.8rem;
      color: #b91c1c;
    }

    .terms {
      margin-top: 0.5rem;
      background: rgba(148, 163, 184, 0.12);
      border-radius: 16px;
      padding: 0.9rem 1rem;
      border: 1px solid rgba(148, 163, 184, 0.2);
    }

    .terms-label {
      display: grid;
      grid-template-columns: auto 1fr;
      align-items: start;
      gap: 0.75rem;
      font-size: 0.85rem;
      color: #475569;
      cursor: pointer;
    }

    .terms-label input {
      width: 18px;
      height: 18px;
      margin-top: 2px;
      accent-color: #6366f1;
    }

    .terms-link {
      color: #4f46e5;
      font-weight: 600;
      text-decoration: none;
    }

    .terms-link:hover {
      text-decoration: underline;
    }

    .form-separator {
      display: flex;
      align-items: center;
      gap: 1rem;
      color: #94a3b8;
      font-size: 0.85rem;
      letter-spacing: 0.1em;
      text-transform: uppercase;
      justify-content: center;
    }

    .form-separator::before,
    .form-separator::after {
      content: '';
      flex: 1;
      height: 1px;
      background: rgba(148, 163, 184, 0.3);
    }

    .btn {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      gap: 0.6rem;
      font-weight: 600;
      padding: 0.85rem 1.1rem;
      border-radius: 12px;
      border: 1px solid transparent;
      cursor: pointer;
      font-size: 0.96rem;
      transition: transform 0.2s ease, box-shadow 0.2s ease, background 0.2s ease, border-color 0.2s ease;
      width: 100%;
    }

    .btn-primary {
      background: linear-gradient(135deg, rgba(59, 82, 160, 0.88) 0%, rgba(48, 70, 138, 0.92) 50%, rgba(34, 51, 104, 0.9) 100%);
      color: #f8fafc;
      border: 1px solid rgba(99, 102, 255, 0.35);
      box-shadow: 0 12px 26px rgba(30, 41, 82, 0.3);
    }

    .btn-primary:hover:not(:disabled) {
      transform: translateY(-2px);
      box-shadow: 0 16px 34px rgba(28, 38, 79, 0.35);
    }

    .btn-ghost {
      background: rgba(79, 70, 229, 0.08);
      color: #4338ca;
      border: 1px solid rgba(79, 70, 229, 0.18);
    }

    .btn-ghost:hover:not(:disabled) {
      background: rgba(79, 70, 229, 0.14);
      transform: translateY(-1px);
    }

    .btn:disabled {
      opacity: 0.6;
      cursor: not-allowed;
      transform: none;
      box-shadow: none;
    }

    .btn-label {
      color: #f5f9ff;
      font-weight: 600;
      letter-spacing: 0.04em;
    }

    .btn-spinner {
      font-size: 0.85rem;
      letter-spacing: 0.08em;
      text-transform: uppercase;
      color: #f5f9ff;
    }

    .btn-google {
      background: rgba(255, 255, 255, 0.92);
      border: 1px solid rgba(148, 163, 184, 0.3);
      color: #0f172a;
    }

    .btn-google:hover:not(:disabled) {
      background: rgba(241, 245, 249, 0.9);
      border-color: rgba(148, 163, 184, 0.45);
    }

    .panel-footer {
      text-align: center;
      display: grid;
      gap: 0.5rem;
      font-size: 0.9rem;
      color: #475569;
    }

    .panel-footer p {
      margin: 0;
    }

    .btn-link {
      background: none;
      border: none;
      padding: 0;
      font-size: inherit;
      color: #4f46e5;
      cursor: pointer;
      font-weight: 600;
      text-decoration: none;
    }

    .btn-link:hover {
      text-decoration: underline;
    }

    .btn-link.subtle {
      color: #64748b;
      font-weight: 500;
    }

    @media (max-width: 1024px) {
      .auth-layout {
        grid-template-columns: 1fr;
        gap: 2rem;
      }

      .auth-aside {
        order: 2;
        background: rgba(15, 23, 42, 0.55);
      }

      .auth-panel {
        order: 1;
      }
    }

    @media (max-width: 640px) {
      .auth-page {
        padding: 1.5rem 1.25rem;
      }

      .auth-card {
        padding: 1.8rem;
        border-radius: 20px;
      }

      .auth-aside {
        border-radius: 20px;
        padding: 1.6rem;
      }

      .aside-list li {
        padding-left: 1.4rem;
      }

      .form-input {
        padding: 0.8rem 0.95rem;
      }

      .terms {
        border-radius: 12px;
      }
    }
  `]
})
export class AuthComponent {
  brandTagline = 'Private multi-model AI studio';
  isLoginMode = signal(true);
  isLoading = signal(false);
  errorMessage = signal('');
  showEmailVerification = signal(false);
  verificationEmail = signal('');
  showForgotPassword = signal(false);
  statusMessage = signal('');

  email = '';
  password = '';
  confirmPassword = '';
  displayName = '';
  resetEmail = '';
  termsAgreed = false;

  constructor(private authService: AuthService, private router: Router) {}

  toggleMode() {
    if (this.isLoading()) return;
    this.isLoginMode.update((value) => !value);
    this.errorMessage.set('');
    this.statusMessage.set('');
    this.resetForm();
  }

  async onSubmit() {
    if (this.isLoading()) return;

    this.isLoading.set(true);
    this.errorMessage.set('');
    this.statusMessage.set('');

    try {
      if (this.isLoginMode()) {
        const user = await this.authService.login(this.email, this.password);
        if (!user.emailVerified) {
          this.errorMessage.set('Please verify your email address before signing in');
          return;
        }
        this.router.navigate(['/app']);
      } else {
        if (this.password !== this.confirmPassword) {
          throw new Error('Passwords do not match');
        }
        const result = await this.authService.register(this.email, this.password, this.displayName);
        if (result.requiresEmailVerification) {
          this.verificationEmail.set(this.email);
          this.showEmailVerification.set(true);
          this.resetForm();
        }
      }
    } catch (error: any) {
      console.error('Auth error:', error);
      this.errorMessage.set(error.message || 'An error occurred');
    } finally {
      this.isLoading.set(false);
    }
  }

  async checkEmailVerification() {
    this.isLoading.set(true);
    this.errorMessage.set('');
    this.statusMessage.set('');

    try {
      await this.authService.completeRegistration();
      this.showEmailVerification.set(false);
      this.router.navigate(['/app']);
    } catch (error: any) {
      console.error('Verification error:', error);
      this.errorMessage.set(error.message || 'Email verification failed');
    } finally {
      this.isLoading.set(false);
    }
  }

  async resendEmail() {
    this.isLoading.set(true);
    this.errorMessage.set('');
    this.statusMessage.set('');

    try {
      await this.authService.resendVerificationEmail();
      this.statusMessage.set('Verification email resent! Please check your inbox.');
    } catch (error: any) {
      console.error('Resend email error:', error);
      this.errorMessage.set(error.message || 'Failed to resend email');
    } finally {
      this.isLoading.set(false);
    }
  }

  showForgotPasswordScreen() {
    this.showForgotPassword.set(true);
    this.isLoginMode.set(true);
    this.errorMessage.set('');
    this.statusMessage.set('');
  }

  async sendPasswordReset() {
    if (!this.resetEmail) {
      this.errorMessage.set('Please enter your email address');
      return;
    }

    this.isLoading.set(true);
    this.errorMessage.set('');
    this.statusMessage.set('');

    try {
      await this.authService.sendPasswordReset(this.resetEmail);
      this.statusMessage.set('Password reset email sent! Please check your inbox.');
    } catch (error: any) {
      console.error('Password reset error:', error);
      this.errorMessage.set(error.message || 'Failed to send password reset email');
    } finally {
      this.isLoading.set(false);
    }
  }

  backToLogin() {
    this.showEmailVerification.set(false);
    this.showForgotPassword.set(false);
    this.isLoginMode.set(true);
    this.statusMessage.set('');
    this.resetForm();
  }

  async signInWithGoogle() {
    if (this.isLoading()) return;

    this.isLoading.set(true);
    this.errorMessage.set('');
    this.statusMessage.set('');

    try {
      await this.authService.signInWithGoogle();
      this.router.navigate(['/app']);
    } catch (error: any) {
      console.error('Google sign-in error:', error);
      this.errorMessage.set(error.message || 'Google sign-in failed');
    } finally {
      this.isLoading.set(false);
    }
  }

  private resetForm() {
    this.email = '';
    this.password = '';
    this.confirmPassword = '';
    this.displayName = '';
    this.resetEmail = '';
    this.termsAgreed = false;
    this.errorMessage.set('');
    this.statusMessage.set('');
  }
}
