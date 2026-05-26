import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { LogoComponent } from '../shared/logo/logo.component';

@Component({
  selector: 'app-about',
  standalone: true,
  imports: [CommonModule, RouterModule, LogoComponent],
  template: `
    <div class="about-page">
      <section class="hero">
        <div class="container">
          <div class="hero-grid">
            <div class="hero-copy">
              <app-logo class="hero-logo" theme="light" size="lg" [showTagline]="false"></app-logo>
              <span class="eyebrow">About Deepenc</span>
              <h1 class="hero-title">
                Private, universal AI designed for independent builders who refuse to compromise on security.
              </h1>
              <p class="hero-subtitle">
                Deepenc is the secure control plane for ChatGPT, Claude, and Gemini. Every prompt, response, and file lives inside a hardware-protected enclave, giving you verified privacy without sacrificing capability.
              </p>
              <div class="hero-actions">
                <a routerLink="/auth" class="cta-button primary">Get Started Free</a>
                <a routerLink="/auth" class="cta-button secondary">Sign In</a>
              </div>
              <div class="hero-meta">
                <p class="hero-note">Free trial available. No credit card required.</p>
                <div class="hero-badges">
                  <span class="hero-badge">Verified zero data retention</span>
                  <span class="hero-badge">Azure Confidential Computing</span>
                </div>
              </div>
            </div>
            <div class="hero-panel">
              <div class="metrics-card">
                <h3>What you gain with Deepenc</h3>
                <ul class="metrics-list">
                  <li>
                    <h4>Protect every interaction</h4>
                    <p>Hardware-enforced TEEs, cryptographic attestation, and zero-knowledge encryption ensure complete data isolation.</p>
                  </li>
                  <li>
                    <h4>Streamline AI access</h4>
                    <p>One platform, three industry-leading models&mdash;ChatGPT, Claude, and Gemini.</p>
                  </li>
                  <li>
                    <h4>Compare and choose</h4>
                    <p>Run parallel queries across models to find the optimal response for each use case.</p>
                  </li>
                </ul>
                <div class="metrics">
                  <div class="metric">
                    <span class="metric-value">0</span>
                    <span class="metric-label">Data leaked outside the enclave</span>
                  </div>
                  <div class="metric">
                    <span class="metric-value">3+</span>
                    <span class="metric-label">Enterprise AI providers orchestrated</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section class="section challenge">
        <div class="container narrow">
          <div class="section-heading">
            <span class="eyebrow accent">The AI Dilemma</span>
            <h2>Consumer AI still trades privacy for convenience.</h2>
            <p class="section-subtitle">
              Today's AI landscape forces an impossible choice: consumer apps that train on your data, or enterprise APIs that are complex to implement and manage. You need cutting-edge AI without donating financial records, health data, or proprietary IP to a training corpus. <strong>Deepenc ends the compromise.</strong>
            </p>
            <p class="section-subtitle">
              <strong>End-to-end security, hardware to API.</strong> Every prompt travels an encrypted path: from your device to our hardware-protected TEE, then directly to enterprise AI APIs that contractually prohibit training on your data. The TEE acts as an impenetrable processing vault&mdash;we can't see inside, providers can't reach back, and your data never enters any training pipeline.
            </p>
            <p class="section-subtitle">
              With Deepenc, you can solve key operational and security challenges:
            </p>
          </div>
          <ul class="card-list">
            <li>
              <h3>Consolidate AI Subscriptions</h3>
              <p>Stop juggling multiple AI subscriptions. One contract gives your team unified access to ChatGPT, Claude, and Gemini—with consolidated billing and user management.</p>
            </li>
            <li>
              <h3>Eliminate Redundant Costs</h3>
              <p>Eliminate overlapping features across vendors. Our transparent usage controls and predictable pricing remove the guesswork from AI procurement.</p>
            </li>
            <li>
              <h3>Hardware-enforced privacy</h3>
              <p>Move beyond policy promises to verifiable isolation: data stays inside the enclave and never leaves unencrypted.</p>
            </li>
            <li>
              <h3>Protect sensitive assets</h3>
              <p>Protect what matters most. Source code, customer data, and research IP stay yours with zero-knowledge encryption and verifiable isolation.</p>
            </li>
          </ul>
        </div>
      </section>

      <section class="section solution">
        <div class="container">
          <div class="section-heading">
            <span class="eyebrow accent">One Platform. Total Control.</span>
            <h2>Deepenc orchestrates every layer of secure AI delivery.</h2>
            <p class="section-subtitle">
              Deepenc eliminates the compromise by delivering on three core promises.
            </p>
          </div>
          <div class="pillars">
            <div class="pillar">
              <h3>Confidential by Design</h3>
              <p>Your data is processed exclusively within a hardware-enforced secure enclave (TEE). Our zero-knowledge system makes your conversations cryptographically inaccessible, even to us. This is not a promise, but an architectural guarantee.</p>
            </div>
            <div class="pillar">
              <h3>Universal Access</h3>
              <p>Access all major AI models from one interface. Switch seamlessly between ChatGPT, Claude, and Gemini to compare responses and find the perfect intelligence for any task.</p>
            </div>
            <div class="pillar">
              <h3>Simplified Cost</h3>
              <p>Consolidate your AI expenses into a single, predictable subscription. Real-time dashboards give you instant visibility into spend—no hidden fees, no surprises.</p>
            </div>
          </div>
        </div>
      </section>

      <section class="section security">
        <div class="container">
          <div class="section-heading">
            <span class="eyebrow accent">Privacy Through Technology, Not Promises</span>
            <h2>Verifiable protection from browser to provider.</h2>
            <p class="section-subtitle">
              Other platforms ask you to trust their policies. We provide security you can verify. Deepenc combines three powerful privacy layers for absolute protection.
            </p>
          </div>
          <div class="security-layers">
            <div class="layer">
              <h3>Layer 1: Enterprise-Grade API Access</h3>
              <p>Only official enterprise APIs from OpenAI, Google, and Anthropic are used, backed by contractual commitments against model training.</p>
            </div>
            <div class="layer">
              <h3>Layer 2: TEE Hardware Encryption</h3>
              <p>We wrap everything in a Trusted Execution Environment (TEE) on Azure Confidential Computing. This creates a hardware-isolated "black box" that processes your data.</p>
            </div>
            <div class="layer">
              <h3>Layer 3: Hardware-Backed Attestation</h3>
              <p>Before handling your data, our server cryptographically proves its security through Microsoft Azure Attestation. If verification fails, the service stops immediately.</p>
            </div>
          </div>
          <div class="security-features">
            <h3>Your data is protected by</h3>
            <div class="feature-list">
              <div class="feature">
                <span class="checkmark">✓</span>
                <div>
                  <strong>Hardware-Level Isolation</strong>
                  <p>Enforced via TEEs, keeping your data invisible to the host system and to Deepenc.</p>
                </div>
              </div>
              <div class="feature">
                <span class="checkmark">✓</span>
                <div>
                  <strong>End-to-End Encryption</strong>
                  <p>Elliptic-curve (ECC P-256) plus AES-256-GCM covers every handshake, transit, and rest state.</p>
                </div>
              </div>
              <div class="feature">
                <span class="checkmark">✓</span>
                <div>
                  <strong>Zero-Knowledge Architecture</strong>
                  <p>We cannot see, store, or re-use your conversations—so we can't leak or lose them.</p>
                </div>
              </div>
            </div>
          </div>
          <figure class="diagram architecture-diagram">
            <img src="assets/Deepenc-architecture.png"
                 alt="Deepenc architecture showing encrypted browser sessions, Azure TEE, and enterprise AI providers">
            <figcaption>
              Deepenc architecture: encrypted browser sessions flow into an Azure TEE backend, hardened with hardware security and enterprise AI providers to safeguard your privacy.
            </figcaption>
          </figure>
        </div>
      </section>

      <section class="section features">
        <div class="container">
          <div class="section-heading">
            <span class="eyebrow accent">Designed for a smarter workflow</span>
            <h2>Everything you need to deploy AI responsibly.</h2>
          </div>
          <div class="feature-grid">
            <article class="feature-card">
              <h3>The Ultimate AI Toolkit</h3>
              <p>Access ChatGPT, Claude, and Gemini in one place. Switch models mid-conversation and compare outputs side-by-side to leverage the unique strengths of each AI.</p>
            </article>
            <article class="feature-card">
              <h3>Multi-Model Conversations</h3>
              <p>Tackle complex problems by threading a single conversation across different AIs. Start with Claude for creative drafting, switch to ChatGPT for code generation, and use Gemini for data analysis—all in one seamless flow.</p>
            </article>
            <article class="feature-card">
              <h3>Unified & Intuitive Interface</h3>
              <p>Forget juggling tabs and logins. With single sign-on and a clean, unified design, your focus stays on your work, not on managing tools.</p>
            </article>
          </div>
          <figure class="diagram">
            <img src="assets/Deepenc-multi-chats.png"
                 alt="Deepenc interface showing multi-model chat layout">
            <figcaption>Compare ChatGPT, Claude, and Gemini responses side-by-side in a single workspace.</figcaption>
          </figure>
        </div>
      </section>

      <section class="section comparison">
        <div class="container">
          <div class="section-heading">
            <span class="eyebrow accent">The Deepenc Advantage</span>
            <h2>Everything in one secure command center.</h2>
          </div>
          <div class="comparison-table">
            <table>
              <thead>
                <tr>
                  <th>Feature</th>
                  <th class="deepenc-col">Deepenc</th>
                  <th>Individual AI Providers</th>
                  <th>Other Aggregators</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td>TEE Hardware Encryption</td>
                  <td class="deepenc-col">✅</td>
                  <td>❌</td>
                  <td>❌</td>
                </tr>
                <tr>
                  <td>Zero-Knowledge Privacy</td>
                  <td class="deepenc-col">✅</td>
                  <td>❌</td>
                  <td>❌</td>
                </tr>
                <tr>
                  <td>Access to All 3 Top Models</td>
                  <td class="deepenc-col">✅</td>
                  <td>❌</td>
                  <td>✅</td>
                </tr>
                <tr>
                  <td>Guaranteed Zero Data Training</td>
                  <td class="deepenc-col">✅ <span class="highlight">(By Default)</span></td>
                  <td>⚠️ (APIs Only, Not Consumer Apps)</td>
                  <td>❔ (Unclear)</td>
                </tr>
                <tr>
                  <td>Unified Interface</td>
                  <td class="deepenc-col">✅</td>
                  <td>❌</td>
                  <td>✅</td>
                </tr>
                <tr>
                  <td>Real-Time Usage Analytics</td>
                  <td class="deepenc-col">✅</td>
                  <td>❌</td>
                  <td>❔</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>

      <section class="section use-cases">
        <div class="container">
          <div class="section-heading">
            <span class="eyebrow accent">Who we serve</span>
            <h2>Built for professionals who demand privacy.</h2>
          </div>
          <div class="use-case-grid">
            <article class="use-case">
              <h3>Developers &amp; Tech Teams</h3>
              <p>Build and test with powerful AI without compromising user data or company intellectual property.</p>
            </article>
            <article class="use-case">
              <h3>Researchers &amp; Analysts</h3>
              <p>Analyze sensitive data and compare insights from multiple AI models in a compliant, confidential environment.</p>
            </article>
            <article class="use-case">
              <h3>Enterprises &amp; Security Teams</h3>
              <p>Deploy AI across your organization with the confidence of enterprise-grade encryption and verifiable data isolation.</p>
            </article>
            <article class="use-case">
              <h3>Consultants &amp; Agencies</h3>
              <p>Manage client projects with the absolute assurance that all sensitive information remains completely private.</p>
            </article>
          </div>
        </div>
      </section>

      <section class="section use-cases private-space">
        <div class="container">
          <div class="section-heading">
            <span class="eyebrow accent">Your private workspace</span>
            <h2>A Private Space for Your Most Important Work</h2>
            <p class="section-subtitle">
              Your privacy shouldn't be limited to the office. Deepenc provides a secure enclave for the thoughts and data that matter most.
            </p>
          </div>
          <div class="use-case-grid">
            <article class="use-case">
              <h3>Protect Your Next Big Idea</h3>
              <p>
                Workshop your novel, refine a confidential business plan, or brainstorm a new invention, knowing your intellectual property is sealed in a digital vault and never used for training.
              </p>
            </article>
            <article class="use-case">
              <h3>Navigate Your Career Confidentially</h3>
              <p>
                Practice for a high-stakes salary negotiation, get advice on a difficult work situation, or plan your next career move with the absolute assurance that your professional ambitions remain private.
              </p>
            </article>
            <article class="use-case">
              <h3>Discuss Finances Privately</h3>
              <p>
                Analyze your personal budget, model investment scenarios, or plan for retirement using your real financial data, confident it will never be seen, stored, or linked to your identity.
              </p>
            </article>
            <article class="use-case">
              <h3>Ask Sensitive Questions without a Digital Trail</h3>
              <p>
                Research a medical condition, explore mental health topics, or ask for personal advice without creating a permanent digital footprint that could be tracked or sold.
              </p>
            </article>
          </div>
        </div>
      </section>

      <section class="section support">
        <div class="container">
          <div class="support-grid">
            <div class="support-copy">
              <span class="eyebrow accent">Need help?</span>
              <h2>We’re here for you at every step.</h2>
              <p class="section-subtitle">
                Whether you’re exploring Deepenc, planning a rollout, or looking for implementation guidance, our specialists respond within one business day.
              </p>
              <div class="support-highlights">
                <span>Security reviews &amp; procurement support</span>
                <span>Live onboarding and workspace coaching</span>
                <span>Priority assistance for enterprise plans</span>
              </div>
            </div>
            <div class="support-card">
              <h3>Contact Support</h3>
              <p>Get in touch with our support team for any questions or assistance.</p>
              <a href="mailto:support@deepenc.com" class="support-email">
                📧 support@deepenc.com
              </a>
              <p class="support-note">We typically respond within 24 hours.</p>
            </div>
          </div>
        </div>
      </section>

      <section class="final-cta">
        <div class="container">
          <div class="final-cta-inner">
            <span class="eyebrow accent">Experience the future of private AI</span>
            <h2>Your best ideas deserve the best protection.</h2>
            <p>Stop choosing between capability and confidentiality. Deepenc keeps every conversation private while giving you the freedom to build.</p>
            <div class="cta-actions">
              <a routerLink="/auth" class="cta-button large">Get Started with Deepenc Today</a>
              <p class="cta-note">Free trial available. No credit card required.</p>
            </div>
          </div>
        </div>
      </section>

      <footer class="footer">
        <div class="container">
          <div class="footer-content">
            <div class="footer-brand">
              <app-logo size="sm" [showTagline]="true" taglinePlacement="stacked" [tagline]="footerTagline" [tight]="true"></app-logo>
            </div>
            <div class="footer-links">
              <a routerLink="/terms">Terms of Service</a>
              <a routerLink="/privacy">Privacy Policy</a>
              <a href="mailto:support@deepenc.com">Support</a>
            </div>
          </div>
          <div class="footer-bottom">
            <p>&copy; 2025 Deepenc. All rights reserved.</p>
          </div>
        </div>
      </footer>
    </div>
  `,
  styles: [`
    * {
      margin: 0;
      padding: 0;
      box-sizing: border-box;
    }

    :host {
      display: block;
    }

    .about-page {
      --color-primary: #4f46e5;
      --color-primary-soft: #eef2ff;
      --color-primary-strong: #312e81;
      --color-surface: #ffffff;
      --color-surface-muted: rgba(255, 255, 255, 0.55);
      --color-body: #0f172a;
      --color-muted: #64748b;
      --color-border: rgba(79, 70, 229, 0.16);
      --color-divider: rgba(15, 23, 42, 0.08);
      font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      color: var(--color-body);
      background:
        radial-gradient(140% 140% at 15% 20%, rgba(99, 102, 241, 0.12), transparent 55%),
        radial-gradient(110% 110% at 80% 0%, rgba(14, 116, 144, 0.08), transparent 60%),
        #f8fafc;
      line-height: 1.65;
    }

    section {
      padding: clamp(4rem, 8vw, 6rem) 0;
    }

    .container {
      width: min(1100px, calc(100% - 3rem));
      margin: 0 auto;
    }

    .container.narrow {
      width: min(880px, calc(100% - 3rem));
    }

    .eyebrow {
      display: inline-flex;
      align-items: center;
      gap: 0.5rem;
      font-size: 0.85rem;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.12em;
      color: var(--color-primary-strong);
    }

    .eyebrow::before {
      content: '';
      width: 20px;
      height: 2px;
      background: var(--color-primary);
      border-radius: 999px;
    }

    .eyebrow.accent {
      color: var(--color-primary-strong);
    }

    .section-heading {
      text-align: center;
      max-width: 700px;
      margin: 0 auto clamp(3rem, 6vw, 4rem);
      display: grid;
      gap: 1.25rem;
    }

    .section-heading h2 {
      font-size: clamp(2.1rem, 3.8vw, 2.8rem);
      font-weight: 700;
      letter-spacing: -0.02em;
      color: var(--color-body);
    }

    .section-subtitle {
      font-size: 1.05rem;
      color: var(--color-muted);
    }

    .hero {
      position: relative;
      padding-top: clamp(6rem, 10vw, 8rem);
    }

    .hero-grid {
      display: grid;
      gap: clamp(2rem, 5vw, 4rem);
      grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
      background: var(--color-surface-muted);
      backdrop-filter: blur(22px);
      border-radius: 28px;
      padding: clamp(2.5rem, 5vw, 3.75rem);
      border: 1px solid rgba(255, 255, 255, 0.6);
      box-shadow:
        0 30px 60px rgba(15, 23, 42, 0.08),
        0 8px 20px rgba(79, 70, 229, 0.08);
      position: relative;
      overflow: hidden;
    }

    .hero-grid::after {
      content: '';
      position: absolute;
      inset: 0;
      background: linear-gradient(135deg, rgba(79, 70, 229, 0.18), transparent 50%);
      mix-blend-mode: lighten;
      pointer-events: none;
    }

    .hero-copy {
      position: relative;
      z-index: 1;
      display: grid;
      gap: 1.5rem;
    }

    .hero-title {
      font-size: clamp(2.6rem, 4vw, 3.5rem);
      font-weight: 700;
      line-height: 1.08;
      letter-spacing: -0.03em;
      color: var(--color-body);
    }

    .hero-subtitle {
      font-size: 1.1rem;
      color: var(--color-muted);
      max-width: 560px;
    }

    .hero-actions {
      display: flex;
      flex-wrap: wrap;
      gap: 0.85rem;
      align-items: center;
    }

    .cta-button {
      padding: 0.95rem 2.1rem;
      border-radius: 999px;
      text-decoration: none;
      font-weight: 600;
      font-size: 1rem;
      transition: transform 0.2s ease, box-shadow 0.2s ease, background 0.2s ease, color 0.2s ease;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      position: relative;
      overflow: hidden;
      cursor: pointer;
    }

    .cta-button.primary {
      background: radial-gradient(120% 120% at 10% 10%, #818cf8 0%, #4f46e5 100%);
      color: #ffffff;
      box-shadow: 0 12px 25px rgba(79, 70, 229, 0.28);
    }

    .cta-button.primary:hover {
      transform: translateY(-2px) scale(1.01);
      box-shadow: 0 16px 32px rgba(79, 70, 229, 0.32);
    }

    .cta-button.secondary {
      background: rgba(79, 70, 229, 0.08);
      color: var(--color-primary-strong);
      border: 1px solid rgba(79, 70, 229, 0.16);
    }

    .cta-button.secondary:hover {
      background: rgba(79, 70, 229, 0.14);
      transform: translateY(-2px);
    }

    .cta-button.large {
      padding: 1.15rem 3rem;
      font-size: 1.05rem;
      box-shadow: 0 15px 35px rgba(79, 70, 229, 0.25);
    }

    .hero-meta {
      display: grid;
      gap: 1.25rem;
    }

    .hero-note {
      color: var(--color-muted);
      font-size: 0.95rem;
    }

    .hero-badges {
      display: flex;
      flex-wrap: wrap;
      gap: 0.75rem;
    }

    .hero-badge {
      font-size: 0.85rem;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.08em;
      background: rgba(79, 70, 229, 0.12);
      color: var(--color-primary-strong);
      padding: 0.45rem 0.9rem;
      border-radius: 999px;
      border: 1px solid rgba(79, 70, 229, 0.18);
    }

    .hero-panel {
      position: relative;
      z-index: 1;
    }

    .metrics-card {
      background: rgba(255, 255, 255, 0.7);
      border-radius: 22px;
      padding: clamp(2rem, 4vw, 2.75rem);
      border: 1px solid rgba(255, 255, 255, 0.6);
      box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.6), 0 25px 40px rgba(15, 23, 42, 0.12);
      display: grid;
      gap: 1.5rem;
    }

    .metrics-card h3 {
      font-size: 1.15rem;
      font-weight: 600;
      color: var(--color-primary-strong);
    }

    .metrics-list {
      display: grid;
      gap: 0.85rem;
      list-style: none;
      color: var(--color-muted);
      font-size: 0.98rem;
    }

    .metrics-list li {
      position: relative;
      padding-left: 1.4rem;
    }

    .metrics-list li::before {
      content: '';
      position: absolute;
      left: 0;
      top: 0.55rem;
      width: 8px;
      height: 8px;
      border-radius: 50%;
      background: linear-gradient(135deg, rgba(79, 70, 229, 0.9), rgba(129, 140, 248, 0.9));
      box-shadow: 0 0 0 3px rgba(79, 70, 229, 0.15);
    }

    .metrics {
      display: grid;
      gap: 1.25rem;
    }

    .metric {
      display: grid;
      gap: 0.25rem;
    }

    .metric-value {
      font-size: 1.8rem;
      font-weight: 700;
      color: var(--color-body);
      letter-spacing: -0.02em;
    }

    .metric-label {
      font-size: 0.95rem;
      color: var(--color-muted);
    }

    .card-list {
      list-style: none;
      display: grid;
      gap: 1.5rem;
    }

    .card-list li {
      background: rgba(255, 255, 255, 0.75);
      border-radius: 18px;
      padding: 1.9rem 2.1rem;
      border: 1px solid rgba(255, 255, 255, 0.7);
      box-shadow: 0 18px 25px rgba(15, 23, 42, 0.08);
      display: grid;
      gap: 0.65rem;
    }

    .card-list h3 {
      font-size: 1.2rem;
      font-weight: 600;
      color: var(--color-body);
    }

    .card-list p {
      color: var(--color-muted);
      font-size: 0.98rem;
    }

    .pillars {
      display: grid;
      gap: 1.75rem;
      grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
    }

    .pillar {
      background: rgba(255, 255, 255, 0.7);
      border-radius: 24px;
      padding: 2.5rem 2.25rem;
      border: 1px solid rgba(255, 255, 255, 0.7);
      box-shadow:
        0 20px 35px rgba(15, 23, 42, 0.08),
        inset 0 1px 0 rgba(255, 255, 255, 0.5);
      display: grid;
      gap: 1rem;
      position: relative;
    }

    .pillar::after {
      content: '';
      position: absolute;
      inset: 0;
      background: linear-gradient(160deg, rgba(79, 70, 229, 0.08), transparent 60%);
      border-radius: inherit;
      pointer-events: none;
    }

    .pillar-icon {
      width: 3.5rem;
      height: 3.5rem;
      border-radius: 1.75rem;
      background: rgba(79, 70, 229, 0.1);
      display: inline-flex;
      align-items: center;
      justify-content: center;
      font-size: 1.6rem;
      color: var(--color-primary-strong);
      position: relative;
      z-index: 1;
    }

    .pillar h3 {
      font-size: 1.35rem;
      font-weight: 600;
      color: var(--color-body);
      position: relative;
      z-index: 1;
    }

    .pillar p {
      color: var(--color-muted);
      line-height: 1.7;
      position: relative;
      z-index: 1;
    }

    .security-layers {
      display: grid;
      gap: 1.5rem;
      grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
      margin-bottom: clamp(2.5rem, 6vw, 3rem);
    }

    .layer {
      background: rgba(255, 255, 255, 0.75);
      border-radius: 20px;
      padding: 2.25rem 2.1rem;
      border: 1px solid rgba(255, 255, 255, 0.7);
      box-shadow: 0 18px 25px rgba(15, 23, 42, 0.08);
      display: grid;
      gap: 0.75rem;
    }

    .layer h3 {
      font-size: 1.15rem;
      font-weight: 600;
      color: var(--color-body);
    }

    .layer p {
      color: var(--color-muted);
      font-size: 0.98rem;
    }

    .security-features {
      margin-bottom: clamp(2.5rem, 6vw, 4rem);
    }

    .security-features h3 {
      text-align: center;
      font-size: 1.25rem;
      font-weight: 600;
      color: var(--color-body);
      margin-bottom: 1.75rem;
    }

    .feature-list {
      display: grid;
      gap: 1.2rem;
      max-width: 660px;
      margin: 0 auto;
    }

    .feature {
      display: grid;
      grid-template-columns: auto 1fr;
      gap: 0.85rem;
      align-items: start;
      background: rgba(255, 255, 255, 0.7);
      border-radius: 18px;
      padding: 1.4rem 1.65rem;
      border: 1px solid rgba(255, 255, 255, 0.7);
      box-shadow: 0 14px 20px rgba(15, 23, 42, 0.08);
    }

    .feature strong {
      display: block;
      font-size: 1.05rem;
      margin-bottom: 0.35rem;
      color: var(--color-body);
    }

    .feature p {
      color: var(--color-muted);
      font-size: 0.95rem;
    }

    .checkmark {
      color: var(--color-primary);
      font-weight: 700;
      font-size: 1.35rem;
      line-height: 1;
      margin-top: 0.2rem;
    }

    .diagram {
      margin: clamp(2.5rem, 6vw, 3.5rem) auto 0;
      max-width: 960px;
      text-align: center;
      background: rgba(15, 23, 42, 0.02);
      border-radius: 24px;
      padding: clamp(1.8rem, 5vw, 2.4rem);
      border: 1px solid rgba(148, 163, 184, 0.2);
      box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.6), 0 18px 32px rgba(15, 23, 42, 0.12);
    }

    .diagram img {
      width: 100%;
      height: auto;
      border-radius: 16px;
      border: 1px solid rgba(148, 163, 184, 0.3);
      box-shadow: 0 24px 40px rgba(15, 23, 42, 0.12);
    }

    .architecture-diagram {
      max-width: 760px;
    }

    .architecture-diagram img {
      width: 82%;
      max-width: 640px;
      margin: 0 auto;
      display: block;
    }

    .diagram figcaption {
      margin-top: 1.2rem;
      font-size: 0.95rem;
      color: var(--color-muted);
    }

    .feature-grid {
      display: grid;
      gap: 1.7rem;
      grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
    }

    .feature-card {
      background: rgba(255, 255, 255, 0.75);
      border-radius: 20px;
      padding: 2.2rem 2.1rem;
      border: 1px solid rgba(255, 255, 255, 0.7);
      box-shadow:
        0 18px 30px rgba(15, 23, 42, 0.08),
        inset 0 1px 0 rgba(255, 255, 255, 0.55);
      display: grid;
      gap: 0.85rem;
    }

    .feature-card h3 {
      font-size: 1.25rem;
      font-weight: 600;
      color: var(--color-body);
    }

    .feature-card p {
      color: var(--color-muted);
      font-size: 0.98rem;
    }

    .comparison-table {
      overflow-x: auto;
      margin-top: 2.5rem;
      border-radius: 22px;
      border: 1px solid rgba(255, 255, 255, 0.6);
      background: rgba(255, 255, 255, 0.75);
      box-shadow: 0 20px 35px rgba(15, 23, 42, 0.08);
    }

    table {
      width: 100%;
      border-collapse: collapse;
      min-width: 720px;
    }

    th, td {
      padding: 1.1rem 1.4rem;
      text-align: left;
      border-bottom: 1px solid rgba(148, 163, 184, 0.3);
      font-size: 0.96rem;
      color: var(--color-body);
    }

    th {
      background: rgba(79, 70, 229, 0.08);
      text-transform: uppercase;
      font-size: 0.75rem;
      letter-spacing: 0.08em;
      color: var(--color-primary-strong);
    }

    tbody tr:last-child td {
      border-bottom: none;
    }

    .deepenc-col {
      background: rgba(79, 70, 229, 0.12);
      font-weight: 600;
      color: var(--color-primary-strong);
    }

    .highlight {
      background: rgba(79, 70, 229, 0.18);
      color: var(--color-primary-strong);
      padding: 0.3rem 0.5rem;
      border-radius: 999px;
      font-size: 0.75rem;
      font-weight: 600;
      letter-spacing: 0.08em;
    }

    .use-case-grid {
      display: grid;
      gap: 1.7rem;
      grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
    }

    .use-case {
      background: rgba(255, 255, 255, 0.75);
      border-radius: 20px;
      padding: 2rem 1.9rem;
      border: 1px solid rgba(255, 255, 255, 0.7);
      box-shadow: 0 18px 25px rgba(15, 23, 42, 0.08);
      display: grid;
      gap: 0.75rem;
    }

    .use-case h3 {
      font-size: 1.15rem;
      font-weight: 600;
      color: var(--color-body);
      letter-spacing: -0.01em;
    }

    .use-case p {
      color: var(--color-muted);
      font-size: 0.95rem;
    }

    .support {
      padding-bottom: clamp(4.5rem, 9vw, 6.5rem);
    }

    .support-grid {
      background: rgba(255, 255, 255, 0.78);
      border-radius: 26px;
      padding: clamp(2.4rem, 5vw, 3.4rem);
      border: 1px solid rgba(255, 255, 255, 0.7);
      box-shadow:
        0 24px 35px rgba(15, 23, 42, 0.1),
        inset 0 1px 0 rgba(255, 255, 255, 0.6);
      display: grid;
      gap: clamp(2rem, 5vw, 3rem);
      grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
    }

    .support-copy {
      display: grid;
      gap: 1rem;
    }

    .support-highlights {
      display: grid;
      gap: 0.75rem;
      font-size: 0.95rem;
      color: var(--color-muted);
    }

    .support-highlights span {
      display: inline-flex;
      align-items: center;
      gap: 0.5rem;
    }

    .support-card {
      background: rgba(79, 70, 229, 0.08);
      border-radius: 20px;
      padding: 2rem 1.8rem;
      border: 1px solid rgba(79, 70, 229, 0.18);
      box-shadow: 0 18px 30px rgba(79, 70, 229, 0.12);
      display: grid;
      gap: 0.75rem;
    }

    .support-card h3 {
      font-size: 1.2rem;
      font-weight: 600;
      color: var(--color-primary-strong);
    }

    .support-card p {
      color: rgba(49, 46, 129, 0.85);
      font-size: 0.97rem;
    }

    .support-email {
      display: inline-flex;
      align-items: center;
      gap: 0.4rem;
      font-weight: 600;
      color: var(--color-primary-strong);
      letter-spacing: 0.02em;
      text-decoration: none;
      font-size: 1rem;
      transition: transform 0.2s ease, color 0.2s ease;
    }

    .support-email:hover {
      color: #3730a3;
      transform: translateY(-1px);
    }

    .support-note {
      color: rgba(49, 46, 129, 0.76);
      font-size: 0.9rem;
    }

    .final-cta {
      text-align: center;
      background: radial-gradient(120% 120% at 50% 0%, rgba(79, 70, 229, 0.18), transparent 70%);
      padding: clamp(5rem, 10vw, 6.5rem) 0;
    }

    .final-cta-inner {
      background: rgba(79, 70, 229, 0.1);
      border-radius: 28px;
      padding: clamp(3rem, 7vw, 4rem);
      border: 1px solid rgba(79, 70, 229, 0.15);
      box-shadow: 0 30px 45px rgba(79, 70, 229, 0.15);
      display: grid;
      gap: 1.5rem;
    }

    .final-cta h2 {
      font-size: clamp(2.1rem, 3.5vw, 2.8rem);
      font-weight: 700;
      color: var(--color-primary-strong);
    }

    .final-cta p {
      color: rgba(49, 46, 129, 0.8);
      font-size: 1.05rem;
      max-width: 640px;
      margin: 0 auto;
    }

    .cta-actions {
      display: grid;
      gap: 0.75rem;
      justify-items: center;
    }

    .cta-note {
      color: rgba(49, 46, 129, 0.7);
      font-size: 0.9rem;
    }

    .footer {
      background: rgba(248, 250, 252, 0.92);
      color: var(--color-body);
      padding: 3.5rem 0 1.5rem;
      border-top: 1px solid rgba(148, 163, 184, 0.28);
    }

    .footer-content {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 2.5rem;
      gap: 1.5rem;
      flex-wrap: wrap;
    }

    .hero-logo {
      display: inline-flex;
      margin-bottom: 1.5rem;
    }

    .footer-brand app-logo {
      display: inline-flex;
    }

    .footer-links {
      display: inline-flex;
      gap: 1.8rem;
      flex-wrap: wrap;
    }

    .footer-links a {
      color: var(--color-muted);
      text-decoration: none;
      font-weight: 500;
      font-size: 0.95rem;
      transition: color 0.2s ease;
    }

    .footer-links a:hover {
      color: var(--color-body);
    }

    .footer-bottom {
      text-align: center;
      padding-top: 2rem;
      border-top: 1px solid rgba(148, 163, 184, 0.25);
      color: var(--color-muted);
      font-size: 0.9rem;
    }

    @media (max-width: 992px) {
      .hero-grid {
        padding: 2.4rem;
      }
    }

    @media (max-width: 768px) {
      .container {
        width: calc(100% - 2.4rem);
      }

      .hero-grid {
        padding: 2rem;
      }

      .metrics-card {
        padding: 2rem;
      }

      .cta-button {
        width: 100%;
        max-width: 280px;
      }

      .cta-button.large {
        max-width: none;
      }

      .hero-badges {
        flex-direction: column;
        align-items: flex-start;
      }

      .card-list li,
      .pillar,
      .layer,
      .feature,
      .feature-card,
      .use-case {
        padding: 1.75rem 1.6rem;
      }

      .comparison-table {
        border-radius: 18px;
      }

      .support-grid {
        padding: 2rem 1.8rem;
      }

      .final-cta-inner {
        padding: 2.5rem 2rem;
      }

      .footer-content {
        flex-direction: column;
        align-items: flex-start;
      }
    }

    @media (max-width: 540px) {
      section {
        padding: 3rem 0;
      }

      .hero-grid {
        padding: 1.6rem;
      }

      .hero-title {
        font-size: 2.2rem;
      }

      .metrics-card {
        padding: 1.8rem 1.6rem;
      }

      .metrics {
        gap: 1rem;
      }

      .diagram {
        padding: 1.6rem;
      }

      .diagram img {
        border-radius: 12px;
      }

      .support-card {
        padding: 1.8rem 1.5rem;
      }

      .footer {
        padding: 3rem 0 1.2rem;
      }
    }
  `]
})
export class AboutComponent {
  footerTagline = 'Private multi-model AI studio';
}
