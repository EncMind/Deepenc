import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { LogoComponent } from '../shared/logo/logo.component';

@Component({
  selector: 'app-landing',
  standalone: true,
  imports: [CommonModule, RouterModule, LogoComponent],
  template: `
    <div class="landing-page">
      <!-- Hero Section -->
      <section class="hero">
        <div class="container">
          <div class="hero-content">
            <div class="hero-logo">
              <app-logo size="lg" theme="light" [tagline]="heroTagline"></app-logo>
            </div>
            <p class="hero-subtitle">
              The unified gateway to ChatGPT, Claude, and Gemini, wrapped in a hardware protected security environment. 
              Your conversations are processed in a hardware-level enclave, guaranteeing they are never seen, stored, or used for model training.
            </p>
            <div class="hero-actions">
              <a routerLink="/auth" class="cta-button primary">Get Started Free</a>
              <a routerLink="/auth" class="cta-button secondary">Sign In</a>
            </div>
            <p class="hero-note">Free trial available. No credit card required.</p>
          </div>
        </div>
      </section>

      <!-- Problem Section -->
      <section class="problem">
        <div class="container">
          <h2>The AI Dilemma</h2>
          <p class="section-subtitle">
            You need the world's best AI, but that means juggling multiple platforms, navigating complex privacy policies, 
            and risking exposure of sensitive company data. You're forced to compromise between leading-edge capability and ironclad security. 
            <strong>Until now.</strong>
          </p>
          
          <div class="problem-grid">
            <div class="problem-item">
              <div class="problem-icon">🔀</div>
              <h3>Managing multiple AI subscriptions</h3>
            </div>
            <div class="problem-item">
              <div class="problem-icon">💸</div>
              <h3>Paying for overlapping features</h3>
            </div>
            <div class="problem-item">
              <div class="problem-icon">❓</div>
              <h3>Uncertain data privacy policies</h3>
            </div>
            <div class="problem-item">
              <div class="problem-icon">⚠️</div>
              <h3>Risk of sensitive data exposure</h3>
            </div>
          </div>
        </div>
      </section>

      <!-- Solution Section -->
      <section class="solution">
        <div class="container">
          <h2>One Platform. Total Control.</h2>
          <p class="section-subtitle">
            Deepenc eliminates the compromise by delivering on three core promises.
          </p>
          
          <div class="pillars">
            <div class="pillar">
              <div class="pillar-icon">🔒</div>
              <h3>Unbreakable Privacy</h3>
              <p>Our platform is built on a zero-knowledge principle. Military-grade TEE encryption creates a secure enclave your data never leaves, ensuring not even we can access your conversations.</p>
            </div>
            
            <div class="pillar">
              <div class="pillar-icon">🌟</div>
              <h3>Universal Access</h3>
              <p>Access all major AI models from one interface. Switch seamlessly between ChatGPT, Claude, and Gemini to compare responses and find the perfect intelligence for any task.</p>
            </div>
            
            <div class="pillar">
              <div class="pillar-icon">💰</div>
              <h3>Simplified Cost</h3>
              <p>Consolidate your AI expenses into a single, predictable subscription. Get transparent, real-time usage tracking with no hidden fees, offering a clear economic advantage over managing individual plans.</p>
            </div>
          </div>
        </div>
      </section>

      <!-- Security Deep Dive -->
      <section class="security">
        <div class="container">
          <h2>Privacy Through Technology, Not Promises</h2>
          <p class="section-subtitle">
            Other platforms ask you to trust their policies. We provide security you can verify. 
            Deepenc combines two powerful privacy layers for absolute protection.
          </p>
          
          <div class="security-layers">
            <div class="layer">
              <h3>Layer 1: Enterprise-Grade API Access</h3>
              <p>We only use the official enterprise APIs from OpenAI, Google, and Anthropic, which contractually forbid the use of your data for model training.</p>
            </div>
            
            <div class="layer">
              <h3>Layer 2: TEE Hardware Encryption</h3>
              <p>We wrap everything in a Trusted Execution Environment (TEE) on Azure Confidential Computing. This creates a hardware-isolated "black box" that processes your data.</p>
            </div>
          </div>
          
          <div class="security-features">
            <h3>Your Data is Protected By:</h3>
            <div class="feature-list">
              <div class="feature">
                <span class="checkmark">✓</span>
                <strong>Hardware-Level Isolation:</strong> Via TEEs, your data is invisible to the host system and to us.
              </div>
              <div class="feature">
                <span class="checkmark">✓</span>
                <strong>End-to-End Encryption:</strong> Using elite ECC P-256 + AES-256-GCM ciphers.
              </div>
              <div class="feature">
                <span class="checkmark">✓</span>
                <strong>Zero-Knowledge Architecture:</strong> We can't see your data, so we can't use, share, or lose it.
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- Features Section -->
      <section class="features">
        <div class="container">
          <h2>Designed for a Smarter Workflow</h2>
          
          <div class="feature-grid">
            <div class="feature-card">
              <h3>The Ultimate AI Toolkit</h3>
              <p>Access ChatGPT, Claude, and Gemini in one place. Switch models mid-conversation and compare outputs side-by-side to leverage the unique strengths of each AI.</p>
            </div>
            
            <div class="feature-card">
              <h3>Multi-Model Conversations</h3>
              <p>Tackle complex problems by threading a single conversation across different AIs. Start with Claude for creative drafting, switch to ChatGPT for code generation, and use Gemini for data analysis—all in one seamless flow.</p>
            </div>
            
            <div class="feature-card">
              <h3>Unified & Intuitive Interface</h3>
              <p>Forget juggling tabs and logins. With single sign-on and a clean, unified design, your focus stays on your work, not on managing tools.</p>
            </div>
          </div>
        </div>
      </section>

      <!-- Comparison Table -->
      <section class="comparison">
        <div class="container">
          <h2>The Deepenc Advantage</h2>
          
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
              </tbody>
            </table>
          </div>
        </div>
      </section>

      <!-- Use Cases -->
      <section class="use-cases">
        <div class="container">
          <h2>Built for Professionals Who Demand Privacy</h2>
          
          <div class="use-case-grid">
            <div class="use-case">
              <h3>👩‍💻 Developers & Tech Teams</h3>
              <p>Build and test with powerful AI without compromising user data or company intellectual property.</p>
            </div>
            
            <div class="use-case">
              <h3>🔬 Researchers & Analysts</h3>
              <p>Analyze sensitive data and compare insights from multiple AI models in a compliant, confidential environment.</p>
            </div>
            
            <div class="use-case">
              <h3>🏢 Enterprises & Security Teams</h3>
              <p>Deploy AI across your organization with the confidence of enterprise-grade encryption and verifiable data isolation.</p>
            </div>
            
            <div class="use-case">
              <h3>🤝 Consultants & Agencies</h3>
              <p>Manage client projects with the absolute assurance that all sensitive information remains completely private.</p>
            </div>
          </div>
        </div>
      </section>

      <!-- Final CTA -->
      <section class="final-cta">
        <div class="container">
          <h2>Experience the Future of Private AI</h2>
          <p>Your best ideas deserve the best protection. Stop choosing and start creating with confidence.</p>
          
          <div class="cta-actions">
            <a routerLink="/auth" class="cta-button large">Get Started with Deepenc Today</a>
            <p class="cta-note">Free trial available. No credit card required.</p>
          </div>
        </div>
      </section>

      <!-- Footer -->
      <footer class="footer">
        <div class="container">
          <div class="footer-content">
            <div class="footer-brand">
              <app-logo size="sm" theme="light" [tagline]="footerTagline" [tight]="true"></app-logo>
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

    .landing-page {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      line-height: 1.6;
      color: #333;
    }

    .container {
      max-width: 1200px;
      margin: 0 auto;
      padding: 0 2rem;
    }

    /* Hero Section */
    .hero {
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      color: white;
      padding: 6rem 0;
      text-align: center;
    }

    .hero-logo {
      display: flex;
      justify-content: center;
      margin-bottom: 1.75rem;
    }

    .hero-subtitle {
      font-size: 1.25rem;
      margin-bottom: 2.5rem;
      max-width: 800px;
      margin-left: auto;
      margin-right: auto;
      color: #e2e8f0;
    }

    .hero-actions {
      display: flex;
      gap: 1rem;
      justify-content: center;
      margin-bottom: 1rem;
    }

    .cta-button {
      padding: 1rem 2rem;
      border-radius: 8px;
      text-decoration: none;
      font-weight: 600;
      font-size: 1.1rem;
      transition: all 0.3s ease;
      display: inline-block;
    }

    .cta-button.primary {
      background: white;
      color: #667eea;
    }

    .cta-button.primary:hover {
      background: #f8f9fa;
      transform: translateY(-2px);
    }

    .cta-button.secondary {
      background: transparent;
      color: white;
      border: 2px solid white;
    }

    .cta-button.secondary:hover {
      background: white;
      color: #667eea;
    }

    .cta-button.large {
      padding: 1.25rem 3rem;
      font-size: 1.2rem;
      background: #667eea;
      color: white;
    }

    .cta-button.large:hover {
      background: #5a6fd8;
      transform: translateY(-2px);
    }

    .hero-note {
      color: #cbd5e0;
      font-size: 0.9rem;
    }

    /* Sections */
    section {
      padding: 4rem 0;
    }

    section h2 {
      font-size: 2.5rem;
      font-weight: 700;
      text-align: center;
      margin-bottom: 1.5rem;
      color: #2d3748;
    }

    .section-subtitle {
      font-size: 1.2rem;
      text-align: center;
      margin-bottom: 3rem;
      max-width: 800px;
      margin-left: auto;
      margin-right: auto;
      color: #4a5568;
    }

    /* Problem Section */
    .problem {
      background: #f8f9fa;
    }

    .problem-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
      gap: 2rem;
      margin-top: 3rem;
    }

    .problem-item {
      text-align: center;
      padding: 2rem;
      background: white;
      border-radius: 12px;
      box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
    }

    .problem-icon {
      font-size: 3rem;
      margin-bottom: 1rem;
    }

    .problem-item h3 {
      font-size: 1.1rem;
      color: #e53e3e;
    }

    /* Solution Section */
    .pillars {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
      gap: 3rem;
      margin-top: 3rem;
    }

    .pillar {
      text-align: center;
      padding: 2.5rem;
      background: white;
      border-radius: 12px;
      box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
    }

    .pillar-icon {
      font-size: 4rem;
      margin-bottom: 1.5rem;
    }

    .pillar h3 {
      font-size: 1.5rem;
      margin-bottom: 1rem;
      color: #2d3748;
    }

    .pillar p {
      color: #4a5568;
      line-height: 1.7;
    }

    /* Security Section */
    .security {
      background: #f8f9fa;
      color: #333;
    }

    .security h2,
    .security h3 {
      color: #2d3748;
    }

    .security-layers {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
      gap: 2rem;
      margin: 3rem 0;
    }

    .layer {
      padding: 2rem;
      background: white;
      border-radius: 12px;
      box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
    }

    .layer h3 {
      margin-bottom: 1rem;
      color: #2d3748;
    }

    .security-features {
      margin-top: 3rem;
    }

    .security-features h3 {
      text-align: center;
      margin-bottom: 2rem;
    }

    .feature-list {
      display: flex;
      flex-direction: column;
      gap: 1rem;
      max-width: 600px;
      margin: 0 auto;
    }

    .feature {
      display: flex;
      align-items: flex-start;
      gap: 1rem;
    }

    .checkmark {
      color: #48bb78;
      font-weight: bold;
      font-size: 1.2rem;
    }

    /* Features Section */
    .features {
      background: #f8f9fa;
    }

    .feature-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
      gap: 2rem;
      margin-top: 3rem;
    }

    .feature-card {
      padding: 2.5rem;
      background: white;
      border-radius: 12px;
      box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
    }

    .feature-card h3 {
      font-size: 1.3rem;
      margin-bottom: 1rem;
      color: #2d3748;
    }

    .feature-card p {
      color: #4a5568;
      line-height: 1.7;
    }

    /* Comparison Table */
    .comparison-table {
      overflow-x: auto;
      margin-top: 2rem;
    }

    table {
      width: 100%;
      border-collapse: collapse;
      background: white;
      border-radius: 12px;
      overflow: hidden;
      box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
    }

    th, td {
      padding: 1rem;
      text-align: center;
      border-bottom: 1px solid #e2e8f0;
    }

    th {
      background: #f8f9fa;
      font-weight: 600;
      color: #2d3748;
    }

    .deepenc-col {
      background: #e6fffa !important;
      font-weight: 600;
    }

    .highlight {
      background: #667eea;
      color: white;
      padding: 0.25rem 0.5rem;
      border-radius: 4px;
      font-size: 0.8rem;
    }

    /* Use Cases */
    .use-cases {
      background: #f8f9fa;
    }

    .use-case-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
      gap: 2rem;
      margin-top: 3rem;
    }

    .use-case {
      padding: 2rem;
      background: white;
      border-radius: 12px;
      box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
    }

    .use-case h3 {
      font-size: 1.2rem;
      margin-bottom: 1rem;
      color: #2d3748;
    }

    .use-case p {
      color: #4a5568;
      line-height: 1.7;
    }

    /* Final CTA */
    .final-cta {
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      color: white;
      text-align: center;
    }

    .final-cta h2 {
      color: white;
      margin-bottom: 1rem;
    }

    .final-cta p {
      font-size: 1.2rem;
      margin-bottom: 2rem;
      color: #e2e8f0;
    }

    .cta-actions {
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 1rem;
    }

    .cta-note {
      color: #cbd5e0;
      font-size: 0.9rem;
    }

    /* Footer */
    .footer {
      background: #2d3748;
      color: white;
      padding: 3rem 0 1rem;
    }

    .footer-content {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 2rem;
    }

    .footer-brand app-logo {
      display: inline-flex;
    }

    .footer-links {
      display: flex;
      gap: 2rem;
    }

    .footer-links a {
      color: #a0aec0;
      text-decoration: none;
      transition: color 0.3s ease;
    }

    .footer-links a:hover {
      color: white;
    }

    .footer-bottom {
      text-align: center;
      padding-top: 2rem;
      border-top: 1px solid #4a5568;
      color: #a0aec0;
    }

    /* Responsive Design */
    @media (max-width: 768px) {
      .hero-logo app-logo .logo-name {
        font-size: 2.6rem;
      }

      .hero-logo app-logo .logo-tagline {
        font-size: 1rem;
      }

      .hero-subtitle {
        font-size: 1.1rem;
      }

      .hero-actions {
        flex-direction: column;
        align-items: center;
      }

      .cta-button {
        width: 100%;
        max-width: 300px;
      }

      section h2 {
        font-size: 2rem;
      }

      .container {
        padding: 0 1rem;
      }

      .security-layers {
        grid-template-columns: 1fr;
      }

      .footer-content {
        flex-direction: column;
        gap: 2rem;
        text-align: center;
      }

      .footer-links {
        justify-content: center;
      }
    }
  `]
})
export class LandingComponent {
  heroTagline = 'Private multi-model AI studio';
  footerTagline = 'Private multi-model AI studio';
}
