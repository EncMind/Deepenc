import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';

type LogoTheme = 'dark' | 'light';
type LogoSize = 'sm' | 'md' | 'lg';
type TaglinePlacement = 'stacked' | 'inline';

@Component({
  selector: 'app-logo',
  standalone: true,
  imports: [CommonModule, RouterModule],
  template: `
    <ng-template #logoContent>
      <span class="logo-mark" aria-hidden="true">
        <svg viewBox="0 0 36 36" role="presentation">
          <defs>
            <radialGradient id="markGlow" cx="18" cy="18" r="16" gradientUnits="userSpaceOnUse">
              <stop offset="0" stop-color="#3e63b8" />
              <stop offset="0.5" stop-color="#5c88d6" />
              <stop offset="1" stop-color="#71c8d7" />
            </radialGradient>
            <linearGradient id="shieldFill" x1="12" y1="8" x2="24" y2="28" gradientUnits="userSpaceOnUse">
              <stop offset="0" stop-color="#f8fbff" />
              <stop offset="1" stop-color="#d6e3ff" />
            </linearGradient>
            <linearGradient id="shieldStroke" x1="11" y1="7" x2="25" y2="29" gradientUnits="userSpaceOnUse">
              <stop offset="0" stop-color="#94b8ff" stop-opacity="0.82" />
              <stop offset="1" stop-color="#4f88d6" stop-opacity="0.42" />
            </linearGradient>
            <linearGradient id="aiText" x1="14" y1="12" x2="24" y2="24" gradientUnits="userSpaceOnUse">
              <stop offset="0" stop-color="#6274f0" />
              <stop offset="0.45" stop-color="#7793f6" />
              <stop offset="1" stop-color="#6ec7f4" />
            </linearGradient>
          </defs>
          <g transform="translate(18, 18) translate(0, -1) scale(1.35) translate(-18, -18)">
            <path
              d="M18 9.2L27 12.3V18.5C27 24 23.3 29.4 18 31.9C12.7 29.4 9 24 9 18.5V12.3L18 9.2Z"
              fill="url(#shieldFill)"
              stroke="url(#shieldStroke)"
              stroke-width="1.1"
            />
            <text
              x="18"
              y="22"
              text-anchor="middle"
              font-family="'Inter', sans-serif"
              font-size="10.5"
              font-weight="800"
              letter-spacing="0.08em"
              fill="url(#aiText)"
              stroke="#050a1a"
              stroke-opacity="0.45"
              stroke-width="0.32"
              paint-order="stroke"
            >AI</text>
          </g>
        </svg>
      </span>
      <div class="logo-stack">
        <span class="logo-type">
          <span class="logo-name">Deepenc</span>
        </span>
        <span *ngIf="showTagline" class="logo-tagline">{{ tagline }}</span>
      </div>
    </ng-template>

    <a
      *ngIf="routerLink; else staticLogo"
      [routerLink]="routerLink"
      class="logo"
      [ngClass]="[themeClass, sizeClass, taglinePlacementClass, tight ? 'logo-tight' : '']"
      [attr.aria-label]="ariaLabel"
    >
      <ng-container *ngTemplateOutlet="logoContent"></ng-container>
    </a>

    <ng-template #staticLogo>
      <div
        class="logo"
        [ngClass]="[themeClass, sizeClass, taglinePlacementClass, tight ? 'logo-tight' : '']"
        [attr.aria-label]="ariaLabel"
      >
        <ng-container *ngTemplateOutlet="logoContent"></ng-container>
      </div>
    </ng-template>
  `,
  styles: [`
    :host {
      display: inline-flex;
      margin: 0;
      padding: 0;
    }

    .logo {
      display: inline-flex;
      align-items: center;
      gap: 0.45rem;
      font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
      text-decoration: none;
      color: inherit;
    }

    .logo-tight {
      gap: 0.55rem;
    }

    .logo-mark {
      width: 3.4rem;
      height: 3.4rem;
      border-radius: 50%;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      position: relative;
      background:
        radial-gradient(135% 135% at 50% 16%, rgba(255, 255, 255, 0.33) 0%, transparent 58%),
        linear-gradient(150deg, #4a59e6 0%, #5d73f1 40%, #73a2ff 100%);
      box-shadow:
        0 12px 22px rgba(28, 36, 102, 0.32),
        inset 0 1px 0 rgba(255, 255, 255, 0.2);
    }

    .logo-mark svg {
      width: 82%;
      height: 82%;
    }

    .logo-stack {
      display: grid;
      gap: 0.2rem;
    }

    .logo-type {
      display: inline-flex;
      align-items: baseline;
      gap: 0.3rem;
    }

    .logo-name {
      font-weight: 700;
      letter-spacing: -0.02em;
      font-size: 1.6rem;
      background: linear-gradient(120deg, #c4c7ff 0%, #9aa7ff 45%, #67e8f9 100%);
      -webkit-background-clip: text;
      background-clip: text;
      -webkit-text-fill-color: transparent;
    }

    .theme-dark .logo-name {
      background: linear-gradient(120deg, #4338ca 0%, #6366f1 45%, #0ea5e9 100%);
      -webkit-background-clip: text;
      background-clip: text;
      -webkit-text-fill-color: transparent;
    }

    .theme-light .logo-name {
      background: linear-gradient(120deg, #e0e7ff 0%, #c4b5fd 45%, #67e8f9 100%);
      -webkit-background-clip: text;
      background-clip: text;
      -webkit-text-fill-color: transparent;
    }
    .logo-tagline {
      font-size: 0.62rem;
      font-weight: 600;
      letter-spacing: 0.14em;
      text-transform: uppercase;
      color: rgba(15, 23, 42, 0.65);
    }

    .theme-dark .logo-tagline {
      color: rgba(71, 85, 105, 0.78);
    }

    .theme-light .logo-tagline {
      color: rgba(226, 232, 240, 0.82);
    }

    .tagline-inline .logo-stack {
      align-items: flex-start;
    }

    .tagline-hidden .logo-type {
      display: inline-flex;
      gap: 0;
    }

    .tagline-hidden .logo-tagline {
      display: none;
    }

    .size-sm .logo-mark {
      width: 1.9rem;
      height: 1.9rem;
      border-radius: 0.8rem;
    }

    .size-sm .logo-name {
      font-size: 1.15rem;
    }

    .size-sm .logo-tagline {
      font-size: 0.72rem;
    }

    .size-lg .logo-mark {
      width: 3rem;
      height: 3rem;
      border-radius: 1.2rem;
    }

    .size-lg .logo-name {
      font-size: 1.9rem;
    }

    .size-lg .logo-tagline {
      font-size: 0.85rem;
    }

    .theme-light .logo {
      color: #f8fafc;
    }

    .theme-light .logo-mark {
      background: linear-gradient(145deg, rgba(74, 222, 128, 0.85), rgba(59, 130, 246, 0.92));
      box-shadow:
        0 16px 32px rgba(15, 23, 42, 0.35),
        inset 0 1px 0 rgba(255, 255, 255, 0.25);
    }

    .theme-light .logo-name {
      background: linear-gradient(120deg, #e0e7ff 0%, #c4b5fd 45%, #67e8f9 100%);
      -webkit-background-clip: text;
      background-clip: text;
      -webkit-text-fill-color: transparent;
    }

    .theme-light .logo-tagline {
      color: rgba(241, 245, 249, 0.78);
    }
  `]
})
export class LogoComponent {
  @Input() routerLink?: string;
  @Input() theme: LogoTheme = 'dark';
  @Input() size: LogoSize = 'md';
  @Input() showTagline = true;
  @Input() tagline = 'Private multi-model AI studio';
  @Input() taglinePlacement: TaglinePlacement = 'stacked';
  @Input() tight = false;

  get themeClass(): string {
    return `theme-${this.theme}`;
  }

  get sizeClass(): string {
    return `size-${this.size}`;
  }

  get taglinePlacementClass(): string {
    if (!this.showTagline) {
      return 'tagline-hidden';
    }
    return `tagline-${this.taglinePlacement}`;
  }

  get ariaLabel(): string {
    return this.showTagline ? `Deepenc — ${this.tagline}` : 'Deepenc';
  }
}
